package onnx

// Package onnx provides local embedding generation using ONNX Runtime.
// It loads pre-trained transformer models and performs tokenization + inference
// to transform text into vector embeddings. The package supports concurrent
// use via configurable worker pools and is designed for high-throughput
// batch processing.
//
// The standard model is all-MiniLM-L6-v2, which produces 384-dimensional
// embeddings optimized for semantic similarity.
import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/DONAR-0/cmdChroma/internal"
	"github.com/daulet/tokenizers"
	ort "github.com/yalue/onnxruntime_go"
)

var (
	MustClose = internal.CheckDefer
)

// EmbedderInterface defines the contract for embedding text into vectors.
type EmbedderInterface interface {
	Embed(text string) ([]float32, error)
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	Close()
}

// EmbedderOption configures an Embedder.
type EmbedderOption func(*Embedder)

// WithNumWorkers sets the number of parallel workers for batch embedding.
// If not set, defaults to the number of CPU cores.
func WithNumWorkers(n int) EmbedderOption {
	return func(e *Embedder) { e.numWorkers = n }
}

// Embedder generates vector embeddings for text using an ONNX model.
// It is safe for concurrent use by multiple goroutines. The embedder
// maintains an ONNX runtime session and a tokenizer instance.
type Embedder struct {
	// session is the ONNX runtime session for model inference.
	// It is initialized once and reused for all embeddings.
	session *ort.DynamicAdvancedSession

	// tokenizer converts raw text into token IDs for the model.
	// Loaded from a tokenizer.json file.
	tokenizer *tokenizers.Tokenizer

	// numWorkers controls parallelism in EmbedDocuments. When processing
	// multiple texts, this many workers run concurrently. Defaults to
	// runtime.NumCPU() if not set via WithNumWorkers.
	numWorkers int
}

// NewEmbedder creates an Embedder instance with the given model, tokenizer,
// and ONNX runtime library. It initializes the ONNX environment and loads
// the model into memory. This operation is expensive; create one embedder
// and reuse it for multiple texts.
//
// Parameters:
//   - modelPath: Path to the model.onnx file
//   - tokenizersPath: Path to the tokenizer.json file
//   - libpath: Path to libonnxruntime.so (ONNX Runtime shared library)
//   - opts: Optional configuration (e.g., WithNumWorkers)
//
// Returns:
//   - *Embedder: Ready to use embedder
//   - error: If model loading, tokenizer loading, or ONNX initialization fails
func NewEmbedder(modelPath, tokenizersPath, libpath string, opts ...EmbedderOption) (*Embedder, error) {
	//1. Setup the ONNX Library
	ort.SetSharedLibraryPath(libpath)

	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("error received when initialize the ONNX Library: %w", err)
	}

	//2. Load dictionary
	tk, err := tokenizers.FromFile(tokenizersPath)
	if err != nil {
		return nil, fmt.Errorf("error received when initialize tokenizers from file: %w", err)
	}

	//3. Load brain
	inputNames := []string{"input_ids", "attention_mask", "token_type_ids"}
	outputNames := []string{"last_hidden_state"}

	sess, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
	if err != nil {
		return nil, fmt.Errorf("error received when starting a session: %w", err)
	}

	e := &Embedder{tokenizer: tk, session: sess, numWorkers: runtime.NumCPU()}

	// Apply options
	for _, opt := range opts {
		opt(e)
	}

	return e, nil
}

// Embed converts a single text into a 384-dimensional vector embedding.
// The text is tokenized, run through the ONNX model, and pooled to produce
// a single vector representing the semantic meaning of the text.
//
// This method is safe for concurrent use.
//
// Returns:
//   - []float32: 384-dimensional embedding (first 384 values from model output)
//   - error: If tokenization or inference fails
func (e *Embedder) Embed(text string) ([]float32, error) {
	// Step A: Tokenize (Text -> IDs)
	ids, _ := e.tokenizer.Encode(text, true)
	// Step B: Prepare tensors (Numbers -> Math Format)
	lenght := int64(len(ids))
	shape := ort.NewShape(1, lenght)

	finalIDs := make([]int64, lenght)
	mask := make([]int64, lenght)
	types := make([]int64, lenght)

	for i, id := range ids {
		finalIDs[i] = int64(id)
		if id != 0 {
			mask[i] = 1
		}

		types[i] = 0
	}

	inT, _ := ort.NewTensor(shape, finalIDs)
	maT, _ := ort.NewTensor(shape, mask)
	tyT, _ := ort.NewTensor(shape, types)

	defer MustClose(inT.Destroy)
	defer MustClose(maT.Destroy)
	defer MustClose(tyT.Destroy)

	// Step C: Run Brain (Math -> Raw Output)
	outT, _ := ort.NewEmptyTensor[float32](ort.NewShape(1, lenght, 384))
	defer MustClose(outT.Destroy)

	err := e.session.Run([]ort.ArbitraryTensor{inT, maT, tyT}, []ort.ArbitraryTensor{outT})
	if err != nil {
		return nil, err
	}

	// Step D: Pooling (Raw Output -> 384 Sentence Vector)
	return outT.GetData()[:384], nil
}

// EmbedDocuments generates embeddings for multiple texts efficiently.
// It automatically chooses between sequential and parallel processing:
//   - If len(texts) < numWorkers: sequential (lower overhead)
//   - Otherwise: parallel using a worker pool of size numWorkers
//
// The order of returned embeddings matches the input order.
//
// The context is used for cancellation; if cancelled, partial results may
// be returned along with the context error.
//
// Returns:
//   - [][]float32: Slice of embeddings, each with 384 dimensions
//   - error: Nil if all texts embedded successfully; non-nil if any fails
func (e *Embedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Small batches: sequential (avoid goroutine overhead)
	if len(texts) < e.numWorkers {
		return e.embedSequential(texts)
	}

	return e.embedParallel(ctx, texts)
}

// embedSequential processes texts one at a time (optimized for small batches).
func (e *Embedder) embedSequential(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text at index %d: %w", i, err)
		}

		results[i] = vec
	}

	return results, nil
}

// embedParallel processes texts concurrently using a worker pool.
func (e *Embedder) embedParallel(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	errors := make([]error, len(texts))

	var wg sync.WaitGroup

	semaphore := make(chan struct{}, e.numWorkers)

	for i, text := range texts {
		wg.Add(1)

		go func(idx int, txt string) {
			defer wg.Done()

			// Acquire semaphore slot (limits concurrency)
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				errors[idx] = ctx.Err()
				return
			}

			// Check context again after acquiring slot
			select {
			case <-ctx.Done():
				errors[idx] = ctx.Err()
				return
			default:
			}

			vec, err := e.Embed(txt)
			if err != nil {
				errors[idx] = err
				return
			}

			results[idx] = vec
		}(i, text)
	}

	wg.Wait()

	// Return first error encountered
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// Close releases all resources held by the embedder, including the tokenizer
// and ONNX runtime session. It is safe to call Close multiple times.
// After Close, the embedder should not be used for further embedding operations.
func (e *Embedder) Close() {
	MustClose(e.tokenizer.Close)
	MustClose(e.session.Destroy)
	MustClose(ort.DestroyEnvironment)
}

package onnx

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
	cd = internal.CheckDefer
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

type Embedder struct {
	session    *ort.DynamicAdvancedSession
	tokenizer  *tokenizers.Tokenizer
	numWorkers int
}

// Embedder initialize the dictionary and the brain
func NewEmbedder(modelPath, tokenizersPath, libpath string, opts ...EmbedderOption) (*Embedder, error) {
	//1. Setup the ONNX Library
	ort.SetSharedLibraryPath(libpath)

	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("error received when initialize the ONNX Library:  %w", err)
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
		return nil, fmt.Errorf("error received when starting a session")
	}

	e := &Embedder{tokenizer: tk, session: sess, numWorkers: runtime.NumCPU()}

	// Apply options
	for _, opt := range opts {
		opt(e)
	}

	return e, nil
}

// Embed converts text into a 384-dimmension vector
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

	defer cd(inT.Destroy)
	defer cd(maT.Destroy)
	defer cd(tyT.Destroy)

	// Step C: Run Brain (Math -> Raw Output)
	outT, _ := ort.NewEmptyTensor[float32](ort.NewShape(1, lenght, 384))
	defer cd(outT.Destroy)

	err := e.session.Run([]ort.ArbitraryTensor{inT, maT, tyT}, []ort.ArbitraryTensor{outT})
	if err != nil {
		return nil, err
	}

	// Step D: Pooling (Raw Output -> 384 Sentence Vector)
	return outT.GetData()[:384], nil
}

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

func (e *Embedder) Close() {
	cd(e.tokenizer.Close)
	cd(e.session.Destroy)
	cd(ort.DestroyEnvironment)
}

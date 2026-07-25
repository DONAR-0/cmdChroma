package llm

/*
#cgo LDFLAGS: -lonnxruntime-genai -lstdc++
#cgo CFLAGS: -I${SRCDIR}/../../models/onnx_runtime/include
#include "ort_genai_c.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type ONNXGenAI struct {
	model     *C.OgaModel
	tokenizer *C.OgaTokenizer
}

func ogError(r *C.OgaResult) error {
	if r == nil {
		return nil
	}
	msg := C.GoString(C.OgaResultGetError(r))
	C.OgaDestroyResult(r)
	return fmt.Errorf("%s", msg)
}

func NewONNXGenAI(modelPath string) (*ONNXGenAI, error) {
	cPath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cPath))

	var model *C.OgaModel
	if err := ogError(C.OgaCreateModel(cPath, &model)); err != nil {
		return nil, fmt.Errorf("OgaCreateModel: %w", err)
	}

	var tokenizer *C.OgaTokenizer
	if err := ogError(C.OgaCreateTokenizer(model, &tokenizer)); err != nil {
		C.OgaDestroyModel(model)
		return nil, fmt.Errorf("OgaCreateTokenizer: %w", err)
	}

	return &ONNXGenAI{model: model, tokenizer: tokenizer}, nil
}

func (o *ONNXGenAI) Close() {
	C.OgaDestroyTokenizer(o.tokenizer)
	C.OgaDestroyModel(o.model)
}

func (o *ONNXGenAI) Generate(prompt string) (string, error) {
	cPrompt := C.CString(prompt)
	defer C.free(unsafe.Pointer(cPrompt))

	var params *C.OgaGeneratorParams
	if err := ogError(C.OgaCreateGeneratorParams(o.model, &params)); err != nil {
		return "", fmt.Errorf("OgaCreateGeneratorParams: %w", err)
	}
	defer C.OgaDestroyGeneratorParams(params)

	cMaxLen := C.CString("max_length")
	C.OgaGeneratorParamsSetSearchNumber(params, cMaxLen, 256)
	C.OgaDestroyString(cMaxLen)

	var gen *C.OgaGenerator
	if err := ogError(C.OgaCreateGenerator(o.model, params, &gen)); err != nil {
		return "", fmt.Errorf("OgaCreateGenerator: %w", err)
	}
	defer C.OgaDestroyGenerator(gen)

	var sequences *C.OgaSequences
	if err := ogError(C.OgaCreateSequences(&sequences)); err != nil {
		return "", fmt.Errorf("OgaCreateSequences: %w", err)
	}
	defer C.OgaDestroySequences(sequences)

	if err := ogError(C.OgaTokenizerEncode(o.tokenizer, cPrompt, sequences)); err != nil {
		return "", fmt.Errorf("OgaTokenizerEncode: %w", err)
	}

	if err := ogError(C.OgaGenerator_AppendTokenSequences(gen, sequences)); err != nil {
		return "", fmt.Errorf("OgaGenerator_AppendTokenSequences: %w", err)
	}

	for !C.OgaGenerator_IsDone(gen) {
		if err := ogError(C.OgaGenerator_GenerateNextToken(gen)); err != nil {
			return "", fmt.Errorf("OgaGenerator_GenerateNextToken: %w", err)
		}
	}

	seqCount := C.OgaGenerator_GetSequenceCount(gen, 0)
	if seqCount == 0 {
		return "", fmt.Errorf("empty output sequence")
	}
	tokens := C.OgaGenerator_GetSequenceData(gen, 0)

	var cText *C.char
	if err := ogError(C.OgaTokenizerDecode(o.tokenizer, tokens, seqCount, &cText)); err != nil {
		return "", fmt.Errorf("OgaTokenizerDecode: %w", err)
	}
	defer C.OgaDestroyString(cText)

	return C.GoString(cText), nil
}

import os
import argparse
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from fastapi.responses import StreamingResponse
import onnxruntime_genai as og

app = FastAPI()

# Global model and tokenizer
model = None
tokenizer = None

class GenerateRequest(BaseModel):
    prompt: str
    model: str  # Path to the model directory

def load_model(model_path: str):
    global model, tokenizer
    try:
        model = og.Model(model_path)
        tokenizer = og.Tokenizer(model)
        print(f"Loaded model from {model_path}")
    except Exception as e:
        print(f"Error loading model: {e}")
        raise e

@app.post("/generate")
async def generate(request: GenerateRequest):
    global model, tokenizer

    # Load model if not loaded or if a different model is requested
    # In a production system, we'd cache models or use a model manager
    # For this sidecar, we load the requested model path.
    try:
        load_model(request.model)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Model load failed: {str(e)}")

    def stream_generator():
        params = og.GeneratorParams(model)
        params.set_input_tokens(tokenizer.encode(request.prompt))

        generator = og.Generator(model, params)

        while not generator.is_done():
            generator.compute_logits()
            token = generator.generate_next_token()
            yield tokenizer.decode([token])

    return StreamingResponse(stream_generator(), media_type="text/plain")

@app.get("/health")
async def health():
    return {"status": "ok"}

if __name__ == "__main__":
    import uvicorn
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=8080)
    args = parser.parse_args()

    uvicorn.run(app, host="0.0.0.0", port=args.port)

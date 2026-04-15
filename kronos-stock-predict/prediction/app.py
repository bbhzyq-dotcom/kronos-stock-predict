import os
import sys
import json
import yaml
import logging
from datetime import datetime
from pathlib import Path

from flask import Flask, request, jsonify
from flask_cors import CORS

sys.path.insert(0, str(Path(__file__).parent))
from kronos_predictor import KronosPredictor

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)

def load_config():
    config_path = Path(__file__).parent / "config.yaml"
    if config_path.exists():
        with open(config_path, 'r') as f:
            return yaml.safe_load(f)
    return {}

config = load_config()
predictor = KronosPredictor()

@app.route('/health', methods=['GET'])
def health():
    return jsonify({"status": "ok", "timestamp": datetime.now().isoformat()})

@app.route('/predict', methods=['POST'])
def predict():
    try:
        data = request.get_json()
        
        code = data.get('code', 'UNKNOWN')
        klines = data.get('klines', [])
        lookbacks = data.get('lookbacks', [120, 30, 10, 5])
        
        if len(klines) < 5:
            return jsonify({
                "error": "insufficient kline data, need at least 5 klines",
                "code": code
            }), 400
        
        logger.info(f"Predicting for {code} with lookbacks: {lookbacks}")
        
        predictions = predictor.predict(klines, lookbacks)
        
        name = code
        if len(klines) > 0:
            name = klines[-1].get('name', code)
        
        formatted_preds = {}
        for lb, pred in predictions.items():
            formatted_preds[lb] = {
                "next_kline": {
                    "open": pred["open"],
                    "high": pred["high"],
                    "low": pred["low"],
                    "close": pred["close"]
                },
                "direction": pred["direction"],
                "change_pct": pred["change_pct"],
                "score": pred["score"]
            }
        
        return jsonify({
            "code": code,
            "name": name,
            "predictions": formatted_preds,
            "timestamp": datetime.now().isoformat()
        })
        
    except Exception as e:
        logger.error(f"Prediction error: {e}")
        return jsonify({"error": str(e)}), 500

@app.route('/model/status', methods=['GET'])
def model_status():
    return jsonify({
        "model_loaded": predictor.model is not None,
        "device": predictor.device,
        "max_context": predictor.max_context
    })

if __name__ == '__main__':
    host = config.get('server', {}).get('host', '0.0.0.0')
    port = config.get('server', {}).get('port', 8081)
    
    logger.info(f"Starting prediction service on {host}:{port}")
    app.run(host=host, port=port, debug=False)

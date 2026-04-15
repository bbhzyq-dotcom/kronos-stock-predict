import os
import sys
import json
import yaml
import logging
from pathlib import Path

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class KronosPredictor:
    def __init__(self, config_path=None):
        self.config = self._load_config(config_path)
        self.model = None
        self.tokenizer = None
        self.device = self.config.get("device", "cuda" if os.path.exists("/proc/driver/nvidia") else "cpu")
        self.max_context = self.config.get("max_context", 512)
        
    def _load_config(self, config_path=None):
        if config_path is None:
            config_path = Path(__file__).parent / "config.yaml"
        if isinstance(config_path, Path):
            config_path = str(config_path)
        if isinstance(config_path, str) and not os.path.exists(config_path):
            return {}
        with open(config_path, 'r') as f:
            return yaml.safe_load(f)
    
    def load_model(self):
        try:
            from transformers import AutoTokenizer, AutoModelForCausalLM
            import torch
            
            model_name = self.config.get("model_name", "NeoQuasar/Kronos-small")
            logger.info(f"Loading Kronos model: {model_name}")
            
            self.tokenizer = AutoTokenizer.from_pretrained(model_name)
            self.model = AutoModelForCausalLM.from_pretrained(
                model_name,
                torch_dtype=torch.float32,
                device_map=self.device
            )
            self.model.eval()
            logger.info(f"Model loaded on {self.device}")
            return True
        except Exception as e:
            logger.warning(f"Failed to load model: {e}, using rule-based prediction")
            self.model = None
            self.tokenizer = None
            return False
    
    def prepare_kline_data(self, klines):
        import pandas as pd
        
        df = pd.DataFrame([{
            'timestamps': k['timestamp'],
            'open': k['open'],
            'high': k['high'],
            'low': k['low'],
            'close': k['close'],
            'volume': k.get('volume', 0),
            'amount': k.get('amount', 0)
        } for k in klines])
        
        df['timestamps'] = pd.to_datetime(df['timestamps'])
        df = df.sort_values('timestamps')
        
        for col in ['open', 'high', 'low', 'close', 'volume', 'amount']:
            if col not in df.columns:
                df[col] = 0
        
        return df
    
    def normalize_klines(self, df):
        price_cols = ['open', 'high', 'low', 'close']
        for col in price_cols:
            if col in df.columns and df[col].iloc[0] != 0:
                df[col] = df[col] / df[col].iloc[0] - 1
        
        if 'volume' in df.columns and df['volume'].iloc[0] != 0:
            df['volume'] = df['volume'] / df['volume'].iloc[0] - 1
        
        return df
    
    def predict_next(self, klines, lookback):
        import pandas as pd
        
        return self._rule_based_prediction(klines)
        
        df = self.prepare_kline_data(klines)
        
        lookback = min(lookback, len(df), self.max_context)
        df_subset = df.tail(lookback).copy()
        
        df_subset = self.normalize_klines(df_subset)
        
        try:
            input_text = self._format_for_kronos(df_subset)
            
            inputs = self.tokenizer(input_text, return_tensors="pt", truncation=True, max_length=self.max_context)
            inputs = {k: v.to(self.device) for k, v in inputs.items()}
            
            with torch.no_grad():
                outputs = self.model.generate(
                    **inputs,
                    max_new_tokens=50,
                    temperature=0.7,
                    top_p=0.9,
                    do_sample=True
                )
            
            response = self.tokenizer.decode(outputs[0], skip_special_tokens=True)
            
            pred = self._parse_kronos_response(response, df_subset)
            return pred
            
        except Exception as e:
            logger.warning(f"Kronos prediction failed, using rule-based: {e}")
            return self._rule_based_prediction(klines)
    
    def _format_for_kronos(self, df):
        ohlcv_str = ""
        for _, row in df.iterrows():
            ohlcv_str += f"{row['open']:.4f},{row['high']:.4f},{row['low']:.4f},{row['close']:.4f},{row.get('volume', 0):.0f};"
        return ohlcv_str
    
    def _parse_kronos_response(self, response, df):
        last_close = df['close'].iloc[-1] if 'close' in df.columns else df.iloc[-1]['close']
        
        try:
            parts = response.strip().split(';')
            if len(parts) >= 1:
                pred_parts = parts[0].split(',')
                if len(pred_parts) >= 4:
                    open_p, high_p, low_p, close_p = [float(x) for x in pred_parts[:4]]
                    
                    open_p = last_close * (1 + open_p)
                    high_p = last_close * (1 + high_p)
                    low_p = last_close * (1 + low_p)
                    close_p = last_close * (1 + close_p)
                    
                    change_pct = (close_p - last_close) / last_close * 100
                    direction = "up" if close_p > last_close else "down"
                    
                    return {
                        "open": round(open_p, 2),
                        "high": round(high_p, 2),
                        "low": round(low_p, 2),
                        "close": round(close_p, 2),
                        "direction": direction,
                        "change_pct": round(change_pct, 2),
                        "score": min(0.95, max(0.3, 0.5 + abs(change_pct) / 100))
                    }
        except:
            pass
        
        return self._rule_based_prediction_from_df(df)
    
    def _rule_based_prediction(self, klines):
        import pandas as pd
        
        df = pd.DataFrame(klines)
        if 'timestamp' in df.columns:
            df['timestamp'] = pd.to_datetime(df['timestamp'])
            df = df.sort_values('timestamp')
        
        return self._rule_based_prediction_from_df(df)
    
    def _rule_based_prediction_from_df(self, df):
        import pandas as pd
        
        if len(df) < 5:
            last_close = df['close'].iloc[-1] if 'close' in df.columns else 100.0
            return {
                "open": last_close,
                "high": last_close,
                "low": last_close,
                "close": last_close,
                "direction": "neutral",
                "change_pct": 0.0,
                "score": 0.5
            }
        
        close = df['close'].values
        volume = df.get('volume', pd.Series([1]*len(df))).values
        
        ma5 = close[-5:].mean() if len(close) >= 5 else close.mean()
        ma10 = close[-10:].mean() if len(close) >= 10 else close.mean()
        ma20 = close[-20:].mean() if len(close) >= 20 else close.mean()
        
        trend = 0
        if ma5 > ma10:
            trend += 1
        if ma10 > ma20:
            trend += 1
        if close[-1] > close[-3]:
            trend += 1
        
        vol_ratio = volume[-1] / (volume[-5:].mean()) if len(volume) >= 5 else 1.0
        
        last_close = close[-1]
        last_high = df['high'].values[-1]
        last_low = df['low'].values[-1]
        
        if trend >= 2 and vol_ratio > 1.2:
            change_pct = 0.5 + (vol_ratio - 1) * 0.3
            direction = "up"
        elif trend <= 1 and vol_ratio < 0.8:
            change_pct = -0.5 - (1 - vol_ratio) * 0.3
            direction = "down"
        else:
            change_pct = (close[-1] - close[-3]) / close[-3] * 50 if len(close) >= 3 else 0
            direction = "up" if change_pct > 0 else "down"
        
        open_p = last_close
        close_p = last_close * (1 + change_pct / 100)
        high_p = max(last_high, close_p) * 1.005
        low_p = min(last_low, close_p) * 0.995
        
        score = min(0.95, max(0.35, 0.5 + abs(change_pct) / 200 + (trend - 2) * 0.05))
        
        return {
            "open": round(open_p, 2),
            "high": round(high_p, 2),
            "low": round(low_p, 2),
            "close": round(close_p, 2),
            "direction": direction,
            "change_pct": round(change_pct, 2),
            "score": round(score, 2)
        }
    
    def predict(self, klines, lookbacks=[120, 30, 10, 5]):
        predictions = {}
        
        for lb in lookbacks:
            if len(klines) >= lb:
                kline_subset = klines[-lb:]
            else:
                kline_subset = klines
            
            pred = self.predict_next(kline_subset, lb)
            predictions[str(lb)] = pred
        
        return predictions

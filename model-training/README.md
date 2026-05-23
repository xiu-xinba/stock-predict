# Fund Model Training

This project trains replacement prediction models for the fund prediction backend.

The Go backend has a reserved model location at `../backend-go/models/`. The
training pipeline exports ONNX classifiers using the contract below so a future
Go ONNX adapter or Python model service can load them consistently:

- input name: `float_input`
- output names: `label`, `probabilities`
- label mapping: `0 = down`, `1 = flat`, `2 = up`
- default feature set: `backend_v1` with 8 numeric features

Two training tasks are supported:

- `next_day`: predicts the next trading day's fund direction.
- `intraday_5m`: predicts the next 5 minutes during trading hours.

## Recommended Environment

Use Python 3.11 or 3.12 for the ML stack. Some ML packages may not yet support
Python 3.14.

```powershell
cd model-training
py -3.11 -m venv .venv
.\.venv\Scripts\Activate.ps1
python -m pip install --upgrade pip
pip install -r requirements.txt
pip install -e .
```

## Data Layout

Put private source data under:

- `data/raw/`
- `data/processed/`

These folders are ignored by git.

## Phase 1 Index-Fund Data Contracts

The index-fund design now starts from a multi-asset data contract instead of a
single fund CSV. The contract covers index funds, tracking indexes, constituents,
stock-index futures, options/volatility, commodities, macro rates, FX,
cross-market signals, capital flow, sentiment, panic factors, labels, and
prediction logs.

Useful commands:

```powershell
python -m fund_model_training.validate_contract --emit-dictionary contracts/index_fund_tables.json
python -m fund_model_training.validate_contract --table fund_daily --emit-header contracts/fund_daily.header.csv
python -m fund_model_training.validate_contract --table fund_daily --csv data/processed/fund_daily.csv
```

The validator checks required columns, datetime parseability, and the most basic
point-in-time rule: `available_time` must not be earlier than source event time.

New planning configs:

- `configs/index_fund_tournament.example.yml`: candidate models and ablation
  plan for daily/weekly and intraday model tournaments.
- `configs/continuous_learning.example.yml`: retraining, drift detection,
  challenger/shadow, and rollback workflow.

## Phase 2 Data Collection

Development data collectors write normalized CSVs that match the phase-1
contracts. AkShare is optional so production data can later be swapped to Wind,
iFinD, exchange-authorized feeds, or broker feeds without changing the training
contracts.

Install optional collector dependencies in a Python 3.11/3.12 environment:

```powershell
pip install -e .[data]
```

Examples:

```powershell
# ETF universe, optionally enriched with fund_code,tracking_index,market
python -m fund_model_training.collect_data --dataset etf_universe --tracking-map data/raw/index_fund_tracking_map.csv --output data/raw/dim_fund.csv

# ETF latest spot snapshot normalized as fund_intraday
python -m fund_model_training.collect_data --dataset etf_spot --output data/raw/fund_intraday_spot.csv

# ETF historical daily rows normalized as fund_daily
python -m fund_model_training.collect_data --dataset etf_daily --symbol 510300 --start-date 20240101 --end-date 20240523 --output data/raw/fund_daily_510300.csv

# A-share index daily rows normalized as index_daily
python -m fund_model_training.collect_data --dataset index_daily --symbol sh000300 --output data/raw/index_daily_sh000300.csv

# Main futures rows normalized as futures_bar
python -m fund_model_training.collect_data --dataset futures_main --symbol IF0 --underlying IF --start-date 20240101 --end-date 20240523 --output data/raw/futures_if0.csv

# Build panic_factor from point-in-time components
python -m fund_model_training.build_panic_factor --input data/raw/panic_components_cn.csv --market CN --output data/processed/panic_factor_cn.csv
```

Collectors validate required columns and basic point-in-time rules before
writing, unless `--skip-validation` is supplied for source debugging.

The panic-factor builder expects component columns such as `iv_component`,
`flow_component`, `news_component`, and `limit_component`. It uses expanding
within-market percentiles, so each row is normalized using only information that
would have been available up to that timestamp.

## Phase 3 Sample Building

Raw contract CSVs are converted into processed training samples before model
training. The first supported sample builder creates daily/weekly index-fund
samples with fund labels, tracking-index features, futures features, and panic
features.

Example:

```powershell
python -m fund_model_training.build_samples `
  --fund-daily data/raw/fund_daily_510300.csv `
  --dim-fund data/raw/dim_fund.csv `
  --index-daily data/raw/index_daily_sh000300.csv `
  --futures data/raw/futures_if0.csv `
  --panic-factor data/processed/panic_factor_cn.csv `
  --tracking-index sh000300 `
  --market CN `
  --output data/processed/daily_weekly_index_fund_samples.csv
```

The output is compatible with the existing trainer:

```powershell
python -m fund_model_training.train --config configs/daily_weekly_index_fund.example.yml
```

For the first model-tournament baseline, use the walk-forward baseline trainer.
It fits both a direction classifier and a return regressor, then writes a report
with naive baselines, high-confidence coverage, and feature importance:

```powershell
python -m fund_model_training.train_baseline `
  --data data/processed/daily_weekly_index_fund_samples.csv `
  --feature-set index_fund_daily_v1 `
  --report-output reports/index_fund_baseline_report.json `
  --metadata-output artifacts/index_fund_baseline_metadata.json `
  --model-output artifacts/index_fund_baseline.joblib
```

Minimum CSV columns:

- `fund_code`
- `asof_time`
- either `label` or a future return column:
  - `future_return_pct_next_day` for `next_day`
  - `future_return_pct_5m` for `intraday_5m`

If feature columns are already present, the trainer uses them directly. If not,
`features.py` can derive a basic feature set from common columns such as NAV,
market index change, sector index change, volume, and flow proxies.

Useful optional columns:

- `estimated_nav`, `latest_nav`, `nav`, `price`, `close`
- `market_change_pct`, `index_change_pct`
- `sector_change_pct`, `industry_change_pct`
- `volume`, `volume_ma20`, `turnover_rate`
- `nav_premium_pct`, `premium_pct`
- `fund_flow_pct`, `etf_flow_pct`, `net_inflow_pct`
- `holding_weighted_change_pct`, `top_holding_change_pct`

Feature sets:

- `backend_v1`: 8 features; compatible with the original API feature contract.
- `extended_v1`: 11 features; includes holding, intraday liquidity, and ETF flow
  proxies. Use it after the backend ONNX input builder is updated to emit the
  same 11 features.

## Train Next-Day Model

```powershell
python -m fund_model_training.train --config configs/next_day.example.yml
```

Default output:

```text
../backend-go/models/model_v1.onnx
```

## Train Intraday 5-Minute Model

```powershell
python -m fund_model_training.train --config configs/intraday_5m.example.yml
```

The current backend does not yet load a second ONNX model for intraday inference,
so this exports to `artifacts/intraday_5m.onnx` by default. After the model is
validated, the backend can be extended to load it separately.

Every export is checked against the backend ONNX contract. If the output names or
probability tensor shape are incompatible, training fails immediately instead of
leaving a model that the API cannot use.

## Accuracy Target

Do not mark a model as meeting 98% accuracy unless rolling walk-forward backtests
prove it on out-of-sample data. For short-horizon financial prediction, a more
realistic production pattern is:

- emit predictions only for high-confidence samples;
- mark the rest as `flat` or `no_signal`;
- track coverage separately from accuracy.

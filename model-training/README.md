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

Use a Conda-managed Python 3.11/3.12 environment for the ML stack, then install
packages with pip inside that environment. Some ML packages may not yet support
Python 3.13/3.14 reliably.

```powershell
cd model-training
conda create -n stock-predict-ml python=3.12 -y
conda run --no-capture-output -n stock-predict-ml python -m pip install --upgrade pip
conda run --no-capture-output -n stock-predict-ml python -m pip install -r requirements.txt
conda run --no-capture-output -n stock-predict-ml python -m pip install -e ".[data,dev]"
```

If `conda` is not on `PATH`, call it by full path, for example
`D:\Miniconda\Scripts\conda.exe run --no-capture-output -n stock-predict-ml ...`.
On Windows, run Conda commands sequentially. Parallel `conda run` calls can
race on Conda's temporary activation files.

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

Install optional collector dependencies in the same Conda environment if they
were not installed with `".[data]"`:

```powershell
conda run --no-capture-output -n stock-predict-ml python -m pip install -e ".[data]"
```

Examples:

```powershell
# End-to-end public-data MVP: collect, build panic proxy, build samples, train baseline
python -m fund_model_training.run_index_fund_pipeline --config configs/public_mvp_pipeline.example.yml

# One-command small batch from a curated universe
python -m fund_model_training.build_index_fund_dataset `
  --universe configs/index_fund_universe.example.csv `
  --start-date 20240101 `
  --end-date 20240523 `
  --raw-dir data/raw/batch `
  --output data/processed/daily_weekly_index_fund_samples.csv `
  --continue-on-error

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

# Build 3/5-minute intraday samples from ETF and index minute bars
python -m fund_model_training.build_intraday_samples `
  --fund-intraday data/raw/fund_intraday_510300.csv `
  --index-intraday data/raw/index_intraday_sh000300.csv `
  --panic-factor data/processed/panic_factor_cn.csv `
  --fund-code 510300 `
  --tracking-index sh000300 `
  --horizon-minutes 5 `
  --output data/processed/intraday_index_fund_samples.csv

# Batch intraday public-data MVP: collect ETF/index minute bars, build samples, train tournament, promote if improved
python -m fund_model_training.run_intraday_pipeline --config configs/public_mvp_intraday_pipeline.example.yml
python -m fund_model_training.run_intraday_pipeline --config configs/public_mvp_intraday_3m_pipeline.example.yml
```

The batch intraday pipeline can maintain an accumulated minute-bar history via
`history_dir`. Running it once per trading day appends new ETF/index minute bars,
deduplicates by symbol and timestamp, and trains from the accumulated history.
Promotion gates currently require at least 3 sampled trading days for
short-horizon models, plus minimum high-confidence accuracy/coverage, so a
single trading day or weak actionable signal cannot replace the current model.

Log predictions and backfill actual labels once future returns are available:

```powershell
python -m fund_model_training.prediction_logging log `
  --prediction-json reports/public_mvp_intraday_prediction_510300.json `
  --output data/processed/prediction_log.csv `
  --append

python -m fund_model_training.prediction_logging backfill `
  --prediction-log data/processed/prediction_log.csv `
  --samples data/processed/public_mvp_intraday_index_fund_samples.csv `
  --output data/processed/prediction_log_backfilled.csv

python -m fund_model_training.prediction_logging evaluate `
  --prediction-log data/processed/prediction_log_backfilled.csv `
  --output reports/prediction_performance_report.json `
  --round-trip-cost-pct 0.02

# Generate a lightweight daily drift report from recent samples
python -m fund_model_training.drift_report `
  --samples data/processed/public_mvp_intraday_index_fund_samples.csv `
  --feature-set index_fund_intraday_v1 `
  --output reports/intraday_5m_drift_report.json

# One-command monitoring cycle: backfill labels, evaluate predictions, report drift
python -m fund_model_training.run_monitoring_cycle --config configs/monitoring_daily.example.yml
python -m fund_model_training.run_monitoring_cycle --config configs/monitoring_intraday_3m.example.yml
python -m fund_model_training.run_monitoring_cycle --config configs/monitoring_intraday_5m.example.yml
```

Prediction logs include both `feature_snapshot_id` and
`feature_snapshot_json`, so a later audit can recover the feature values used by
an individual prediction. Performance reports include a paper-trading block
with mean/cumulative cost-adjusted return, high-confidence return, actionable
return, and win rate after the configured round-trip cost.

Collectors validate required columns and basic point-in-time rules before
writing, unless `--skip-validation` is supplied for source debugging.

The batch dataset builder uses `configs/index_fund_universe.example.csv` as the
curated starting point. Add rows there when expanding from smoke tests to a
larger fund universe. The `futures_underlying` column prevents IF/IH/IC/IM
features from being mixed across unrelated index funds.

The public-data MVP pipeline intentionally uses a proxy panic factor derived
from index volatility, drawdown, futures returns, and open-interest changes. It
is a runnable substitute until licensed option, fund-flow, and sentiment feeds
are connected; do not treat it as the final fear model.

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

For short-horizon 3/5-minute prediction, build intraday samples first and run
the same candidate-model tournament with the intraday feature set:

```powershell
python -m fund_model_training.train_tournament --config configs/index_fund_intraday_tournament.example.yml
```

The intraday tournament configs enable `purged_split` plus a 3/5-minute
`embargo_minutes` setting by default. This removes training rows whose future
label window overlaps the test window, so short-horizon scores are not inflated
by overlapping labels.

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

To run the first real model tournament across multiple tabular candidates:

```powershell
python -m fund_model_training.train_tournament --config configs/index_fund_tournament_train.example.yml
python -m fund_model_training.train_tournament --config configs/index_fund_weekly_tournament.example.yml
```

The tournament currently compares `hist_gbdt`, `random_forest`, `extra_trees`,
`gradient_boosting`, and `logistic_ridge`, then writes a champion `.joblib`
bundle plus a full candidate report. The report can also include ablation
blocks such as `without_panic_factor`, `without_futures_commodity`, and
`without_index_features`, making feature-group contribution visible before a
challenger is promoted. These are still tabular challengers; TFT,
PatchTST/iTransformer, Temporal GNN, and DeepLOB require a larger dataset and a
separate deep-learning training stack.

For index funds, the regression path follows the design report's decomposition
instead of treating fund NAV as an isolated series:

```text
fund_return = tracking_index_return + tracking_error
```

Daily/weekly samples emit `future_index_return_pct_*` and
`future_tracking_error_pct_*`; 3/5-minute samples emit the matching intraday
columns. Baseline and tournament trainers fit component regressors when those
targets are present, then sum the predicted tracking-index return and tracking
error back into the final fund return. Prediction JSON exposes the same
`return_decomposition` block so downstream services can audit the components.

Training reports also include `market_regime` slices for the design-required
state checks: bull/bear/sideways trend, high/low panic, and high/low volatility.
Promotion configs can enable `min_market_regime_slice_rows` and
`min_market_regime_accuracy` so a challenger with an obvious weak market-state
slice is rejected instead of becoming champion.

Training reports also include empirical residual prediction intervals under
`prediction_interval`. The model service returns those calibrated bounds with
`method`, `level`, and empirical coverage so downstream services can report the
design-required interval coverage instead of only a heuristic spread.

Tournament reports now include true rolling walk-forward backtests under each
candidate's `rolling_backtest` block. Candidate selection prefers rolling
average balanced accuracy before the single holdout score, and promotion configs
can require `min_rolling_backtest_folds` plus
`min_rolling_balanced_accuracy` before a challenger is allowed to replace the
current champion.

Baseline and tournament trainers also attempt a classifier-only ONNX sidecar
export next to the champion bundle, for example
`artifacts/public_mvp_index_fund_tournament_champion.classifier.onnx`. The
sidecar uses the backend contract input `float_input` and outputs `label` plus
`probabilities`; export status is recorded in training metadata under
`classifier_onnx`. If `skl2onnx`/`onnxruntime` is unavailable, training
continues and records `status: skipped`. Set `require_classifier_onnx: true` in
a training config when a release gate should fail on ONNX export problems.

Run prediction from the current tournament champion:

```powershell
python -m fund_model_training.predict_model `
  --model artifacts/public_mvp_index_fund_tournament_champion.joblib `
  --samples data/processed/public_mvp_daily_weekly_index_fund_samples.csv `
  --fund-code 510300
```

The output JSON is shaped for backend consumption: fund code/name, model
metadata, next-day direction, predicted change percent, confidence,
`prediction_interval`, range, class probabilities, top factors, and data-quality
flags.

Serve the champion model for the Go backend:

```powershell
python -m fund_model_training.serve_model `
  --model artifacts/public_mvp_index_fund_tournament_champion.joblib `
  --samples data/processed/public_mvp_daily_weekly_index_fund_samples.csv `
  --host 127.0.0.1 `
  --port 8090
```

After a model has been promoted, serving can read the registry pointer directly:

```powershell
python -m fund_model_training.serve_model `
  --registry-current model_registry/daily_index_fund/current.json `
  --host 127.0.0.1 `
  --port 8090 `
  --prediction-log-output data/processed/prediction_log.csv
```

Then set `MODEL_SERVICE_URL=http://127.0.0.1:8090` for `backend-go`. The backend
calls `GET /predict/{fund_code}` and falls back to its Go baseline if the model
service is unavailable.

For a separate weekly champion, serve `model_registry/weekly_index_fund/current.json`
only after `retraining_cycle_weekly` promotes one, then set
`WEEKLY_MODEL_SERVICE_URL` in `backend-go`. The current public weekly challenger
trains successfully but is rejected by the high-confidence accuracy gate. For a
separate 3/5-minute champion, run another `serve_model` instance on a different
port and set `INTRADAY_MODEL_SERVICE_URL`.

Promote a challenger into the local model registry only after it passes
configured gates. The example retraining configs require high-confidence
accuracy as well as balanced accuracy and MAE, and can reject a challenger whose
high-confidence accuracy or coverage regresses against the current champion.
`signal_status=actionable` is emitted only for confident up/down signals from a
model that also passes the online actionability gate: high-confidence accuracy
must be at least 50%, high-confidence coverage must be at least 5%, and
calibration ECE must be no worse than 0.12. Low confidence, flat predictions,
and otherwise weak models remain logged for monitoring but are not treated as
tradeable actions:

```powershell
python -m fund_model_training.promote_model `
  --task daily_index_fund `
  --model artifacts/public_mvp_index_fund_tournament_champion.joblib `
  --metadata artifacts/public_mvp_index_fund_tournament_metadata.json `
  --min-balanced-accuracy 0.34 `
  --min-high-confidence-accuracy 0.50 `
  --max-high-confidence-accuracy-drop 0.0 `
  --max-high-confidence-coverage-drop 0.0

python -m fund_model_training.promote_model `
  --task intraday_5m `
  --model artifacts/public_mvp_index_fund_intraday_tournament_champion.joblib `
  --metadata artifacts/public_mvp_index_fund_intraday_tournament_metadata.json `
  --min-balanced-accuracy 0.34 `
  --max-regression-mae 0.12 `
  --min-high-confidence-accuracy 0.50 `
  --max-high-confidence-accuracy-drop 0.0
```

For scheduled continuous learning, use the retraining cycle runner. It trains a
new challenger, compares it with `model_registry/<task>/current.json`, and only
promotes the challenger if the gates pass. Pair it with the drift report and
prediction-log evaluation above in a scheduled job:

```powershell
python -m fund_model_training.run_retraining_cycle --config configs/retraining_cycle_daily.example.yml
python -m fund_model_training.run_retraining_cycle --config configs/retraining_cycle_weekly.example.yml
python -m fund_model_training.run_retraining_cycle --config configs/retraining_cycle_intraday_5m.example.yml
```

Before promotion, a challenger can be evaluated as a shadow model from
backfilled prediction logs:

```powershell
python -m fund_model_training.shadow_evaluation --config configs/shadow_evaluation_daily.example.yml
```

The shadow report checks labeled row count, shadow run days, direction accuracy,
high-confidence coverage, and cost-adjusted paper return. If a current champion
model version is supplied, the challenger must not underperform it on direction
accuracy or mean cost-adjusted return. When a challenger is promoted, the
previous `current.json` and model file are preserved as `rollback.json` and
`rollback.joblib`.

To restore the rollback alias after a bad deployment:

```powershell
python -m fund_model_training.rollback_model --task daily_index_fund --reason "manual validation failed"
```

The current MVP uses CPU-friendly tabular candidates because the public sample
set is still small. ROCm/GPU training becomes useful when the dataset expands to
multi-month or multi-year minute bars and we add deep sequence candidates such
as PatchTST, iTransformer, TFT, or DeepLOB.

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

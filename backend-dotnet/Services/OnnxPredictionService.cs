using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using Microsoft.ML.OnnxRuntime;
using Microsoft.ML.OnnxRuntime.Tensors;
using StockPredictApi.Configuration;
using StockPredictApi.Dtos;

namespace StockPredictApi.Services;

public class OnnxPredictionService : IDisposable
{
    private InferenceSession? _session;
    private readonly string _modelPath;
    private readonly ILogger<OnnxPredictionService> _logger;
    private bool _modelLoaded;
    private string _inputName = "float_input";

    private static readonly string[] FeatureNames = {
        "momentum_5d", "return_10d", "volatility_20d", "volume_ratio",
        "market_beta", "sector_momentum", "flow_signal", "mean_reversion"
    };

    private static readonly Dictionary<int, string> LabelMap = new()
    {
        { 0, "down" }, { 1, "flat" }, { 2, "up" }
    };

    private static readonly List<Dictionary<string, string>> FactorPool = new()
    {
        new() { { "name", "momentum_5d" }, { "description", "5日动量因子" } },
        new() { { "name", "return_10d" }, { "description", "10日收益率因子" } },
        new() { { "name", "volatility_20d" }, { "description", "20日波动率因子" } },
        new() { { "name", "volume_ratio" }, { "description", "成交量比率因子" } },
        new() { { "name", "market_beta" }, { "description", "市场Beta因子" } },
        new() { { "name", "sector_momentum" }, { "description", "行业动量因子" } },
        new() { { "name", "flow_signal" }, { "description", "资金流向信号" } },
        new() { { "name", "mean_reversion" }, { "description", "均值回归因子" } },
    };

    public OnnxPredictionService(IOptions<AppSettings> settings, ILogger<OnnxPredictionService> logger)
    {
        _modelPath = settings.Value.Model.Path;
        _logger = logger;
        LoadModel();
    }

    private void LoadModel()
    {
        try
        {
            if (File.Exists(_modelPath))
            {
                _session = new InferenceSession(_modelPath);
                _modelLoaded = true;

                // Dynamically get the input name from the model
                var inputMeta = _session.InputMetadata;
                if (inputMeta.Count > 0)
                {
                    _inputName = inputMeta.Keys.First();
                    _logger.LogInformation("ONNX model input name: {InputName}, shape: {Shape}",
                        _inputName, string.Join(",", inputMeta[_inputName].Dimensions));
                }

                var outputMeta = _session.OutputMetadata;
                foreach (var kvp in outputMeta)
                {
                    try
                    {
                        _logger.LogInformation("ONNX model output: {Name}, type: {Type}",
                            kvp.Key, kvp.Value.ElementType);
                    }
                    catch (Exception ex)
                    {
                        _logger.LogInformation("ONNX model output: {Name} (metadata unavailable: {Error})",
                            kvp.Key, ex.Message);
                    }
                }

                _logger.LogModelLoaded(_modelPath);
            }
            else
            {
                _logger.LogModelNotFound(_modelPath);
            }
        }
        catch (Exception ex)
        {
            _logger.LogModelLoadFailed(ex);
        }
    }

    public bool IsModelLoaded() => _modelLoaded;

    public (Direction direction, double confidence, double[] probabilities)? Predict(double[] features)
    {
        if (!_modelLoaded || _session == null) return null;

        try
        {
            var inputTensor = new DenseTensor<float>(features.Select(f => (float)f).ToArray(), [1, features.Length]);
            var inputNamed = new List<NamedOnnxValue> { NamedOnnxValue.CreateFromTensor(_inputName, inputTensor) };

            using var results = _session.Run(inputNamed);
            if (results == null || results.Count == 0) return null;

            // Model outputs: "label" (int64 class index) and optionally "probabilities" (float[])
            int predClass = 0;
            double[] probabilities = [0.33, 0.34, 0.33]; // default uniform

            foreach (var result in results)
            {
                try
                {
                    if (result.Name == "label")
                    {
                        // Label output: single int64 class index
                        var labelData = result.AsEnumerable<long>();
                        if (labelData != null)
                        {
                            var labelValue = labelData.FirstOrDefault();
                            predClass = (int)labelValue;
                        }
                    }
                    else if (result.Name == "probabilities")
                    {
                        // Probability output: float array (if available)
                        var probsData = result.AsEnumerable<float>();
                        if (probsData != null)
                        {
                            var probs = probsData.ToArray();
                            if (probs.Length > 0)
                            {
                                probabilities = probs.Select(p => (double)p).ToArray();
                            }
                        }
                    }
                }
                catch (Exception ex)
                {
                    _logger.LogDebug(ex, "Failed to parse ONNX output: {Name}", result.Name);
                }
            }

            // Validate predClass is within range
            if (predClass < 0 || !LabelMap.ContainsKey(predClass)) return null;

            var directionStr = LabelMap[predClass];
            var direction = directionStr switch
            {
                "up" => Direction.Up,
                "down" => Direction.Down,
                _ => Direction.Flat
            };

            // Confidence: use probability if available, otherwise use 1.0 for deterministic label
            double confidence = probabilities.Length > predClass ? probabilities[predClass] : 0.6;

            return (direction, Math.Round(confidence, 4), probabilities);
        }
        catch (Exception ex)
        {
            _logger.LogPredictionFailed(ex);
            return null;
        }
    }

    public List<FactorItemDto> GetTopFactors(int count)
    {
        // Note: These are reference importance weights, not derived from the model's actual inference.
        // The ONNX model is a 3-class classifier and does not output per-feature importance.
        // These weights represent typical relative importance from domain knowledge.
        double[] importance = { 0.15, 0.14, 0.13, 0.12, 0.12, 0.12, 0.11, 0.11 };
        var indices = Enumerable.Range(0, FactorPool.Count)
            .OrderByDescending(i => importance[i])
            .Take(count)
            .ToList();

        double maxImp = importance[indices[0]];
        return indices.Select(i => new FactorItemDto
        {
            Name = FactorPool[i]["name"],
            Importance = Math.Round(importance[i] / maxImp, 2),
            Description = FactorPool[i]["description"]
        }).ToList();
    }

    public void Dispose()
    {
        _session?.Dispose();
        GC.SuppressFinalize(this);
    }
}

internal static partial class OnnxLogMessages
{
    [LoggerMessage(Level = LogLevel.Information, Message = "ONNX model loaded from {Path}")]
    public static partial void LogModelLoaded(this ILogger logger, string path);

    [LoggerMessage(Level = LogLevel.Warning, Message = "ONNX model file not found at {Path}, using mock predictions")]
    public static partial void LogModelNotFound(this ILogger logger, string path);

    [LoggerMessage(Level = LogLevel.Warning, Message = "Failed to load ONNX model, using mock predictions")]
    public static partial void LogModelLoadFailed(this ILogger logger, Exception ex);

    [LoggerMessage(Level = LogLevel.Warning, Message = "ONNX prediction failed")]
    public static partial void LogPredictionFailed(this ILogger logger, Exception ex);
}

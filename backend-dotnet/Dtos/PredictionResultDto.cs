using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class PredictionResultDto
{
    [JsonPropertyName("direction")] public Direction Direction { get; set; }
    [JsonPropertyName("direction_confidence")] public double DirectionConfidence { get; set; }
    [JsonPropertyName("predicted_change_pct")] public double PredictedChangePct { get; set; }
    [JsonPropertyName("change_range")] public ChangeRangeDto ChangeRange { get; set; } = new();
    [JsonPropertyName("top_factors")] public List<FactorItemDto> TopFactors { get; set; } = new();
    [JsonPropertyName("reliability")] public string Reliability { get; set; } = "model";
    [JsonPropertyName("reliability_note")] public string ReliabilityNote { get; set; } = "";
}

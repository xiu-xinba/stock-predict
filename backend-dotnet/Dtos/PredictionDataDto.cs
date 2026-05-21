using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class PredictionDataDto
{
    [JsonPropertyName("fund_code")] public string FundCode { get; set; } = "";
    [JsonPropertyName("fund_name")] public string FundName { get; set; } = "";
    [JsonPropertyName("prediction")] public PredictionResultDto Prediction { get; set; } = new();
    [JsonPropertyName("market_snapshot")] public MarketSnapshotDto MarketSnapshot { get; set; } = new();
}

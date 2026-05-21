using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class WatchlistItemDto
{
    [JsonPropertyName("fund_code")] public string FundCode { get; set; } = "";
    [JsonPropertyName("fund_name")] public string FundName { get; set; } = "";
    [JsonPropertyName("fund_type")] public string FundType { get; set; } = "";
    [JsonPropertyName("estimated_nav")] public double EstimatedNav { get; set; }
    [JsonPropertyName("change_pct")] public double ChangePct { get; set; }
    [JsonPropertyName("direction")] public Direction Direction { get; set; }
    [JsonPropertyName("added_at")] public long AddedAt { get; set; }
}

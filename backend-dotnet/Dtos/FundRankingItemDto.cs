using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class FundRankingItemDto
{
    [JsonPropertyName("rank")] public int Rank { get; set; }
    [JsonPropertyName("fund_code")] public string FundCode { get; set; } = "";
    [JsonPropertyName("fund_name")] public string FundName { get; set; } = "";
    [JsonPropertyName("fund_type")] public string FundType { get; set; } = "";
    [JsonPropertyName("change_pct")] public double ChangePct { get; set; }
    [JsonPropertyName("estimated_nav")] public double EstimatedNav { get; set; }
}

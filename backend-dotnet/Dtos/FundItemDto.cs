using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class FundItemDto
{
    [JsonPropertyName("fund_code")] public string FundCode { get; set; } = "";
    [JsonPropertyName("fund_name")] public string FundName { get; set; } = "";
    [JsonPropertyName("fund_type")] public string FundType { get; set; } = "";
    [JsonPropertyName("company")] public string? Company { get; set; }
    [JsonPropertyName("manager")] public string? Manager { get; set; }
    [JsonPropertyName("latest_nav")] public double LatestNav { get; set; }
    [JsonPropertyName("cumulative_nav")] public double CumulativeNav { get; set; }
    [JsonPropertyName("return_1m")] public double Return1M { get; set; }
    [JsonPropertyName("return_3m")] public double Return3M { get; set; }
    [JsonPropertyName("return_6m")] public double Return6M { get; set; }
    [JsonPropertyName("return_1y")] public double Return1Y { get; set; }
    [JsonPropertyName("return_3y")] public double Return3Y { get; set; }
    [JsonPropertyName("risk_level")] public string? RiskLevel { get; set; }
    [JsonPropertyName("inception_date")] public string? InceptionDate { get; set; }
}

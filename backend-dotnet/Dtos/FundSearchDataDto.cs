using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class FundSearchDataDto
{
    [JsonPropertyName("items")] public List<FundItemDto> Items { get; set; } = new();
    [JsonPropertyName("total")] public int Total { get; set; }
    [JsonPropertyName("page")] public int Page { get; set; }
    [JsonPropertyName("size")] public int Size { get; set; }
}

public class FundFiltersDto
{
    [JsonPropertyName("types")] public List<string> Types { get; set; } = new();
    [JsonPropertyName("companies")] public List<string> Companies { get; set; } = new();
    [JsonPropertyName("risk_levels")] public List<string> RiskLevels { get; set; } = new();
}

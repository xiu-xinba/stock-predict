using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class MarketIndexDto
{
    [JsonPropertyName("code")] public string Code { get; set; } = "";
    [JsonPropertyName("name")] public string Name { get; set; } = "";
    [JsonPropertyName("market")] public string Market { get; set; } = "cn"; // cn, hk, us
    [JsonPropertyName("value")] public double Value { get; set; }
    [JsonPropertyName("change")] public double Change { get; set; }
    [JsonPropertyName("change_pct")] public double ChangePct { get; set; }
    [JsonPropertyName("high")] public double High { get; set; }
    [JsonPropertyName("low")] public double Low { get; set; }
    [JsonPropertyName("prev_close")] public double PrevClose { get; set; }
    [JsonPropertyName("volume")] public double Volume { get; set; }
    [JsonPropertyName("mini_chart_data")] public List<double> MiniChartData { get; set; } = new();
    [JsonPropertyName("update_time")] public string UpdateTime { get; set; } = "";
    [JsonPropertyName("data_source")] public string DataSource { get; set; } = "real";
}

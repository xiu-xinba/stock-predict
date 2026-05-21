using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace StockPredictApi.Models;

[Table("market_indices")]
public class MarketIndex
{
    [Key]
    [Column("code")]
    [StringLength(10)]
    public string Code { get; set; } = string.Empty;

    [Column("name")]
    [StringLength(20)]
    [Required]
    public string Name { get; set; } = string.Empty;

    [Column("value")]
    public double Value { get; set; } = 0.0;

    [Column("change")]
    public double Change { get; set; } = 0.0;

    [Column("change_pct")]
    public double ChangePct { get; set; } = 0.0;

    [Column("mini_chart_data")]
    public string MiniChartData { get; set; } = "[]";

    [Column("updated_at")]
    public DateTime? UpdatedAt { get; set; }
}

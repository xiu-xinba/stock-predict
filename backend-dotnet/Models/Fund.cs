using System.ComponentModel.DataAnnotations;
using System.ComponentModel.DataAnnotations.Schema;

namespace StockPredictApi.Models;

[Table("funds")]
public class Fund
{
    [Key]
    [Column("fund_code")]
    [StringLength(6)]
    public string FundCode { get; set; } = string.Empty;

    [Column("fund_name")]
    [StringLength(100)]
    [Required]
    public string FundName { get; set; } = string.Empty;

    [Column("fund_type")]
    [StringLength(20)]
    [Required]
    public string FundType { get; set; } = string.Empty;

    [Column("company")]
    [StringLength(100)]
    public string? Company { get; set; }

    [Column("manager")]
    [StringLength(100)]
    public string? Manager { get; set; }

    [Column("inception_date")]
    [StringLength(10)]
    public string? InceptionDate { get; set; }

    [Column("latest_nav")]
    public double LatestNav { get; set; } = 0.0;

    [Column("cumulative_nav")]
    public double CumulativeNav { get; set; } = 0.0;

    [Column("return_1m")]
    public double Return1M { get; set; } = 0.0;

    [Column("return_3m")]
    public double Return3M { get; set; } = 0.0;

    [Column("return_6m")]
    public double Return6M { get; set; } = 0.0;

    [Column("return_1y")]
    public double Return1Y { get; set; } = 0.0;

    [Column("return_3y")]
    public double Return3Y { get; set; } = 0.0;

    [Column("risk_level")]
    [StringLength(10)]
    public string? RiskLevel { get; set; }

    [Column("strategy")]
    [StringLength(500)]
    public string? Strategy { get; set; }

    [Column("estimated_nav")]
    public double EstimatedNav { get; set; } = 0.0;

    [Column("change_pct")]
    public double ChangePct { get; set; } = 0.0;

    [Column("updated_at")]
    public DateTime? UpdatedAt { get; set; }
}

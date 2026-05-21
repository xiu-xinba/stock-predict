using Microsoft.EntityFrameworkCore;
using StockPredictApi.Models;

namespace StockPredictApi.Data;

public class AppDbContext : DbContext
{
    public AppDbContext(DbContextOptions<AppDbContext> options) : base(options) { }

    public DbSet<Fund> Funds => Set<Fund>();
    public DbSet<MarketIndex> MarketIndices => Set<MarketIndex>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        modelBuilder.Entity<Fund>(entity =>
        {
            entity.HasKey(e => e.FundCode);
            entity.Property(e => e.FundCode).HasMaxLength(6);
            entity.Property(e => e.FundName).HasMaxLength(100).IsRequired();
            entity.Property(e => e.FundType).HasMaxLength(20).IsRequired();
            entity.Property(e => e.Company).HasMaxLength(100);
            entity.Property(e => e.Manager).HasMaxLength(100);
            entity.Property(e => e.InceptionDate).HasMaxLength(10);
            entity.Property(e => e.RiskLevel).HasMaxLength(10);
            entity.Property(e => e.Strategy).HasMaxLength(500);

            entity.HasIndex(e => e.FundName);
            entity.HasIndex(e => e.FundType);
            entity.HasIndex(e => e.Company);
            entity.HasIndex(e => e.RiskLevel);
        });

        modelBuilder.Entity<MarketIndex>(entity =>
        {
            entity.HasKey(e => e.Code);
            entity.Property(e => e.Code).HasMaxLength(10);
            entity.Property(e => e.Name).HasMaxLength(20).IsRequired();
        });
    }
}

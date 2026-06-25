using ListeTiime.Core;

namespace ListeTiime.Tests;

public sealed class SettingsStoreTests : IDisposable
{
    private readonly string _tempDirectory = Path.Combine(Path.GetTempPath(), "liste-tiime-settings", Guid.NewGuid().ToString("N"));

    [Fact]
    public void SettingsStore_SavesAndLoadsFormatAndLedgerMode()
    {
        var store = new SettingsStore(_tempDirectory);
        var settings = new UserSettings();
        settings.SetFormats(OutputFormats.Pdf);
        settings.SetLedgerMode(LedgerMode.Customer);

        store.Save(settings);

        var loaded = store.Load(OutputFormats.Excel, LedgerMode.Auto, fixInvalidFormats: false);
        Assert.False(loaded.Excel);
        Assert.True(loaded.Pdf);
        Assert.Equal(LedgerMode.Customer, loaded.ParsedLedgerMode);
    }

    [Fact]
    public void SettingsStore_DirectModeFallsBackToExcelWhenSavedFormatsAreInvalid()
    {
        var store = new SettingsStore(_tempDirectory);
        var settings = new UserSettings();
        settings.SetFormats(OutputFormats.None);
        store.Save(settings);

        var loaded = store.Load(OutputFormats.Excel | OutputFormats.Pdf, LedgerMode.Auto, fixInvalidFormats: true);

        Assert.True(loaded.Excel);
        Assert.False(loaded.Pdf);
    }

    public void Dispose()
    {
        if (Directory.Exists(_tempDirectory))
        {
            Directory.Delete(_tempDirectory, recursive: true);
        }
    }
}

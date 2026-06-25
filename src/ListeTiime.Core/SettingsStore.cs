using System.Text.Json;
using System.Text.Json.Serialization;

namespace ListeTiime.Core;

public sealed class UserSettings
{
    [JsonPropertyName("excel")]
    public bool Excel { get; set; } = true;

    [JsonPropertyName("pdf")]
    public bool Pdf { get; set; }

    [JsonPropertyName("ledger_mode")]
    public string LedgerMode { get; set; } = "auto";

    public OutputFormats Formats => (Excel ? OutputFormats.Excel : OutputFormats.None)
        | (Pdf ? OutputFormats.Pdf : OutputFormats.None);

    public LedgerMode ParsedLedgerMode => LedgerMode.ToLowerInvariant() switch
    {
        "supplier" => ListeTiime.Core.LedgerMode.Supplier,
        "customer" => ListeTiime.Core.LedgerMode.Customer,
        _ => ListeTiime.Core.LedgerMode.Auto
    };

    public void SetFormats(OutputFormats formats)
    {
        Excel = formats.HasFlag(OutputFormats.Excel);
        Pdf = formats.HasFlag(OutputFormats.Pdf);
    }

    public void SetLedgerMode(LedgerMode mode)
    {
        LedgerMode = mode switch
        {
            ListeTiime.Core.LedgerMode.Supplier => "supplier",
            ListeTiime.Core.LedgerMode.Customer => "customer",
            _ => "auto"
        };
    }
}

public sealed class SettingsStore
{
    private static readonly JsonSerializerOptions SerializerOptions = new()
    {
        WriteIndented = true
    };

    private readonly string _settingsPath;

    public SettingsStore(string? baseDirectory = null)
    {
        var directory = baseDirectory
            ?? Path.Combine(
                Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData),
                AppConstants.SettingsDirectoryName);
        _settingsPath = Path.Combine(directory, AppConstants.SettingsFileName);
    }

    public UserSettings Load(OutputFormats defaultFormats, LedgerMode defaultLedgerMode = LedgerMode.Auto, bool fixInvalidFormats = true)
    {
        UserSettings settings;
        if (File.Exists(_settingsPath))
        {
            try
            {
                settings = JsonSerializer.Deserialize<UserSettings>(File.ReadAllText(_settingsPath), SerializerOptions)
                    ?? new UserSettings();
            }
            catch
            {
                settings = new UserSettings();
                settings.SetFormats(defaultFormats);
                settings.SetLedgerMode(defaultLedgerMode);
            }
        }
        else
        {
            settings = new UserSettings();
            settings.SetFormats(defaultFormats);
            settings.SetLedgerMode(defaultLedgerMode);
        }

        if (fixInvalidFormats && settings.Formats == OutputFormats.None)
        {
            settings.SetFormats(OutputFormats.Excel);
        }

        return settings;
    }

    public void Save(UserSettings settings)
    {
        Directory.CreateDirectory(Path.GetDirectoryName(_settingsPath)!);
        File.WriteAllText(_settingsPath, JsonSerializer.Serialize(settings, SerializerOptions));
    }
}

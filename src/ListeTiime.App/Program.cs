using ListeTiime.Core;

namespace ListeTiime.App;

static class Program
{
    [STAThread]
    static int Main(string[] args)
    {
        ApplicationConfiguration.Initialize();

        var noDialog = args.Any(argument => string.Equals(argument, "--no-dialog", StringComparison.OrdinalIgnoreCase));
        var files = args
            .Where(argument => !string.Equals(argument, "--no-dialog", StringComparison.OrdinalIgnoreCase))
            .ToArray();

        if (files.Length > 0)
        {
            return RunDirect(files, noDialog);
        }

        Application.Run(new MainForm());
        return 0;
    }

    private static int RunDirect(IReadOnlyList<string> files, bool noDialog)
    {
        try
        {
            var settings = new SettingsStore().Load(
                OutputFormats.Excel | OutputFormats.Pdf,
                LedgerMode.Auto,
                fixInvalidFormats: true);
            var result = new LedgerProcessor().ProcessFiles(
                files,
                new ProcessingOptions(settings.Formats, settings.ParsedLedgerMode, TreatmentRole.First));

            if (noDialog)
            {
                Console.WriteLine(LedgerProcessor.BuildSummary(result));
            }

            return 0;
        }
        catch (Exception exception)
        {
            var message = exception.Message;
            File.WriteAllText(Path.Combine(Path.GetTempPath(), "relance_tiime_error.log"), exception.ToString());

            if (noDialog)
            {
                Console.Error.WriteLine(message);
            }
            else
            {
                MessageBox.Show(message, "Liste Tiime", MessageBoxButtons.OK, MessageBoxIcon.Error);
            }

            return 1;
        }
    }
}

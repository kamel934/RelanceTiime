using ListeTiime.Core;

namespace ListeTiime.App;

public sealed class MetadataDialog : Form
{
    private readonly TextBox _clientTextBox;
    private readonly TextBox _yearTextBox;

    private MetadataDialog(string path)
    {
        Text = "Nom de fichier non standard";
        StartPosition = FormStartPosition.CenterParent;
        FormBorderStyle = FormBorderStyle.FixedDialog;
        MaximizeBox = false;
        MinimizeBox = false;
        ClientSize = new Size(430, 210);
        Font = new Font("Segoe UI", 10f);

        var layout = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            Padding = new Padding(18),
            ColumnCount = 2,
            RowCount = 4
        };
        layout.ColumnStyles.Add(new ColumnStyle(SizeType.Absolute, 90));
        layout.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        layout.RowStyles.Add(new RowStyle(SizeType.Absolute, 48));
        layout.RowStyles.Add(new RowStyle(SizeType.Absolute, 44));
        layout.RowStyles.Add(new RowStyle(SizeType.Absolute, 44));
        layout.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        Controls.Add(layout);

        var message = new Label
        {
            Text = $"Impossible de déduire le client et l'année depuis : {Path.GetFileName(path)}",
            Dock = DockStyle.Fill,
            AutoEllipsis = true
        };
        layout.Controls.Add(message, 0, 0);
        layout.SetColumnSpan(message, 2);

        layout.Controls.Add(new Label { Text = "Client", Dock = DockStyle.Fill, TextAlign = ContentAlignment.MiddleLeft }, 0, 1);
        _clientTextBox = new TextBox { Dock = DockStyle.Fill };
        layout.Controls.Add(_clientTextBox, 1, 1);

        layout.Controls.Add(new Label { Text = "Année", Dock = DockStyle.Fill, TextAlign = ContentAlignment.MiddleLeft }, 0, 2);
        _yearTextBox = new TextBox { Dock = DockStyle.Fill, Text = DateTime.Today.Year.ToString() };
        layout.Controls.Add(_yearTextBox, 1, 2);

        var buttons = new FlowLayoutPanel
        {
            Dock = DockStyle.Fill,
            FlowDirection = FlowDirection.RightToLeft
        };
        var ok = new Button { Text = "OK", DialogResult = DialogResult.OK, Width = 88 };
        var cancel = new Button { Text = "Annuler", DialogResult = DialogResult.Cancel, Width = 88 };
        buttons.Controls.Add(ok);
        buttons.Controls.Add(cancel);
        layout.Controls.Add(buttons, 0, 3);
        layout.SetColumnSpan(buttons, 2);

        AcceptButton = ok;
        CancelButton = cancel;
    }

    public static FileMetadata? Show(IWin32Window owner, string path)
    {
        using var dialog = new MetadataDialog(path);
        if (dialog.ShowDialog(owner) != DialogResult.OK)
        {
            return null;
        }

        var client = dialog._clientTextBox.Text.Trim().ToUpperInvariant();
        var year = dialog._yearTextBox.Text.Trim();
        if (string.IsNullOrWhiteSpace(client) || string.IsNullOrWhiteSpace(year))
        {
            return null;
        }

        return new FileMetadata(client, year);
    }
}

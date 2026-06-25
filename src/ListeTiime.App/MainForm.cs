using ListeTiime.Core;

namespace ListeTiime.App;

public sealed class MainForm : Form
{
    private readonly SettingsStore _settingsStore = new();
    private readonly LedgerProcessor _processor = new();
    private readonly UserSettings _settings;
    private readonly List<Panel> _dropPanels = [];
    private readonly Dictionary<TreatmentRole, Label> _titles = [];

    private CheckBox _excelCheckBox = null!;
    private CheckBox _pdfCheckBox = null!;
    private RadioButton _autoRadio = null!;
    private RadioButton _supplierRadio = null!;
    private RadioButton _customerRadio = null!;
    private Label _statusTitle = null!;
    private TextBox _statusText = null!;
    private bool _loading;

    public MainForm()
    {
        _settings = _settingsStore.Load(OutputFormats.Excel, LedgerMode.Auto, fixInvalidFormats: false);
        InitializeComponent();
        ApplySettings();
        UpdatePanelTitles();
    }

    private void InitializeComponent()
    {
        Text = "Mise en forme - Grand livre";
        MinimumSize = new Size(820, 700);
        Size = new Size(920, 740);
        StartPosition = FormStartPosition.CenterScreen;
        BackColor = Color.FromArgb(245, 247, 250);
        Font = new Font("Segoe UI", 10f);

        var iconPath = Path.Combine(AppContext.BaseDirectory, "logo_compta_premium.ico");
        if (!File.Exists(iconPath))
        {
            iconPath = Path.GetFullPath(Path.Combine(AppContext.BaseDirectory, "..", "..", "..", "..", "logo_compta_premium.ico"));
        }

        if (File.Exists(iconPath))
        {
            Icon = new Icon(iconPath);
        }

        var main = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            Padding = new Padding(28),
            ColumnCount = 1,
            RowCount = 6,
            BackColor = BackColor
        };
        main.RowStyles.Add(new RowStyle(SizeType.Absolute, 72));
        main.RowStyles.Add(new RowStyle(SizeType.Absolute, 64));
        main.RowStyles.Add(new RowStyle(SizeType.Percent, 42));
        main.RowStyles.Add(new RowStyle(SizeType.Percent, 31));
        main.RowStyles.Add(new RowStyle(SizeType.Absolute, 34));
        main.RowStyles.Add(new RowStyle(SizeType.Percent, 27));
        Controls.Add(main);

        main.Controls.Add(CreateHeader(), 0, 0);
        main.Controls.Add(CreateOptionsPanel(), 0, 1);
        main.Controls.Add(CreateTopDropArea(), 0, 2);
        main.Controls.Add(CreateBottomDropArea(), 0, 3);
        main.Controls.Add(CreateNoteLabel(), 0, 4);
        main.Controls.Add(CreateStatusPanel(), 0, 5);
    }

    private Control CreateHeader()
    {
        var panel = new Panel { Dock = DockStyle.Fill, BackColor = BackColor };
        var title = new Label
        {
            Text = "Mise en forme du grand livre",
            Dock = DockStyle.Top,
            Height = 34,
            Font = new Font("Segoe UI", 18f, FontStyle.Bold),
            ForeColor = Color.FromArgb(15, 23, 42)
        };
        var subtitle = new Label
        {
            Text = "Déposez un export CSV pour générer les listes prêtes à envoyer.",
            Dock = DockStyle.Top,
            Height = 26,
            Font = new Font("Segoe UI", 10f),
            ForeColor = Color.FromArgb(71, 85, 105)
        };
        panel.Controls.Add(subtitle);
        panel.Controls.Add(title);
        return panel;
    }

    private Control CreateOptionsPanel()
    {
        var panel = new FlowLayoutPanel
        {
            Dock = DockStyle.Fill,
            FlowDirection = FlowDirection.LeftToRight,
            WrapContents = false,
            BackColor = BackColor,
            Padding = new Padding(0, 10, 0, 0)
        };

        panel.Controls.Add(CreateGroupLabel("Format"));
        _excelCheckBox = CreateCheckBox("Excel");
        _pdfCheckBox = CreateCheckBox("PDF");
        _excelCheckBox.CheckedChanged += SaveFormatSettings;
        _pdfCheckBox.CheckedChanged += SaveFormatSettings;
        panel.Controls.Add(_excelCheckBox);
        panel.Controls.Add(_pdfCheckBox);

        panel.Controls.Add(CreateSpacer(24));
        panel.Controls.Add(CreateGroupLabel("Grand livre"));
        _autoRadio = CreateRadioButton("Auto");
        _supplierRadio = CreateRadioButton("Fournisseur");
        _customerRadio = CreateRadioButton("Client");
        _autoRadio.CheckedChanged += SaveLedgerMode;
        _supplierRadio.CheckedChanged += SaveLedgerMode;
        _customerRadio.CheckedChanged += SaveLedgerMode;
        panel.Controls.Add(_autoRadio);
        panel.Controls.Add(_supplierRadio);
        panel.Controls.Add(_customerRadio);

        return panel;
    }

    private Control CreateTopDropArea()
    {
        var grid = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            ColumnCount = 2,
            RowCount = 1,
            BackColor = BackColor,
            Padding = new Padding(0, 8, 0, 8)
        };
        grid.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        grid.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 50));
        grid.RowStyles.Add(new RowStyle(SizeType.Percent, 100));

        grid.Controls.Add(CreateDropPanel(TreatmentRole.First, Color.FromArgb(31, 78, 120)), 0, 0);
        grid.Controls.Add(CreateDropPanel(TreatmentRole.Second, Color.FromArgb(138, 106, 19)), 1, 0);
        return grid;
    }

    private Control CreateBottomDropArea()
    {
        var outer = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            ColumnCount = 3,
            RowCount = 1,
            BackColor = BackColor,
            Padding = new Padding(0, 8, 0, 8)
        };
        outer.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 20));
        outer.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 60));
        outer.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 20));
        outer.RowStyles.Add(new RowStyle(SizeType.Percent, 100));
        outer.Controls.Add(CreateDropPanel(TreatmentRole.Both, Color.FromArgb(51, 65, 85)), 1, 0);
        return outer;
    }

    private Panel CreateDropPanel(TreatmentRole role, Color accent)
    {
        var panel = new Panel
        {
            Dock = DockStyle.Fill,
            Margin = new Padding(8),
            BackColor = Color.White,
            Cursor = Cursors.Hand,
            AllowDrop = true,
            Tag = role
        };

        panel.Paint += (_, args) =>
        {
            using var pen = new Pen(accent, 2f);
            args.Graphics.DrawRectangle(pen, 1, 1, panel.Width - 3, panel.Height - 3);
        };
        panel.Click += (_, _) => SelectCsv(role);
        panel.DragEnter += DropPanel_DragEnter;
        panel.DragDrop += (_, args) => DropPanel_DragDrop(role, args);

        var layout = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            ColumnCount = 1,
            RowCount = 5,
            Padding = new Padding(16, 12, 16, 12),
            BackColor = Color.White,
            Cursor = Cursors.Hand
        };
        layout.RowStyles.Add(new RowStyle(SizeType.Percent, 50));
        layout.RowStyles.Add(new RowStyle(SizeType.Absolute, 60));
        layout.RowStyles.Add(new RowStyle(SizeType.Absolute, 10));
        layout.RowStyles.Add(new RowStyle(SizeType.Absolute, 36));
        layout.RowStyles.Add(new RowStyle(SizeType.Percent, 50));
        layout.Click += (_, _) => SelectCsv(role);

        var title = new Label
        {
            Dock = DockStyle.Fill,
            TextAlign = ContentAlignment.MiddleCenter,
            Font = new Font("Segoe UI", 14f, FontStyle.Bold),
            ForeColor = accent,
            Cursor = Cursors.Hand,
            AutoEllipsis = false,
            UseCompatibleTextRendering = true
        };
        title.Click += (_, _) => SelectCsv(role);
        _titles[role] = title;

        var button = new Button
        {
            Text = "Choisir un CSV",
            Anchor = AnchorStyles.None,
            Width = 150,
            Height = 34,
            Cursor = Cursors.Hand,
            BackColor = accent,
            ForeColor = Color.White,
            FlatStyle = FlatStyle.Flat
        };
        button.FlatAppearance.BorderSize = 0;
        button.Click += (_, _) => SelectCsv(role);

        layout.Controls.Add(title, 0, 1);
        layout.Controls.Add(button, 0, 3);
        panel.Controls.Add(layout);
        _dropPanels.Add(panel);
        return panel;
    }

    private Control CreateNoteLabel()
    {
        return new Label
        {
            Dock = DockStyle.Fill,
            Text = "Le dépôt direct d'un CSV sur l'icône du .exe génère la première liste du type détecté.",
            TextAlign = ContentAlignment.MiddleCenter,
            ForeColor = Color.FromArgb(100, 116, 139),
            BackColor = BackColor
        };
    }

    private Control CreateStatusPanel()
    {
        var panel = new Panel
        {
            Dock = DockStyle.Fill,
            BackColor = Color.White,
            Padding = new Padding(16),
            Margin = new Padding(8)
        };
        panel.Paint += (_, args) =>
        {
            using var pen = new Pen(Color.FromArgb(203, 213, 225), 1f);
            args.Graphics.DrawRectangle(pen, 0, 0, panel.Width - 1, panel.Height - 1);
        };

        _statusTitle = new Label
        {
            Text = "Résultat",
            Dock = DockStyle.Top,
            Height = 28,
            Font = new Font("Segoe UI", 11f, FontStyle.Bold),
            ForeColor = Color.FromArgb(15, 23, 42)
        };

        _statusText = new TextBox
        {
            Dock = DockStyle.Fill,
            Multiline = true,
            ReadOnly = true,
            BorderStyle = BorderStyle.None,
            BackColor = Color.White,
            ForeColor = Color.FromArgb(71, 85, 105),
            Text = "Déposez un CSV pour lancer un traitement.",
            ScrollBars = ScrollBars.Vertical
        };

        panel.Controls.Add(_statusText);
        panel.Controls.Add(_statusTitle);
        return panel;
    }

    private static Label CreateGroupLabel(string text)
    {
        return new Label
        {
            Text = text,
            AutoSize = true,
            Font = new Font("Segoe UI", 10f, FontStyle.Bold),
            ForeColor = Color.FromArgb(30, 41, 59),
            Margin = new Padding(0, 5, 8, 0)
        };
    }

    private static CheckBox CreateCheckBox(string text)
    {
        return new CheckBox
        {
            Text = text,
            AutoSize = true,
            Margin = new Padding(0, 4, 14, 0),
            Cursor = Cursors.Hand
        };
    }

    private static RadioButton CreateRadioButton(string text)
    {
        return new RadioButton
        {
            Text = text,
            AutoSize = true,
            Margin = new Padding(0, 4, 14, 0),
            Cursor = Cursors.Hand
        };
    }

    private static Control CreateSpacer(int width)
    {
        return new Label { Width = width, Height = 1 };
    }

    private void ApplySettings()
    {
        _loading = true;
        _excelCheckBox.Checked = _settings.Excel;
        _pdfCheckBox.Checked = _settings.Pdf;
        _autoRadio.Checked = _settings.ParsedLedgerMode == LedgerMode.Auto;
        _supplierRadio.Checked = _settings.ParsedLedgerMode == LedgerMode.Supplier;
        _customerRadio.Checked = _settings.ParsedLedgerMode == LedgerMode.Customer;
        _loading = false;
    }

    private void SaveFormatSettings(object? sender, EventArgs args)
    {
        if (_loading)
        {
            return;
        }

        _settings.Excel = _excelCheckBox.Checked;
        _settings.Pdf = _pdfCheckBox.Checked;
        _settingsStore.Save(_settings);
    }

    private void SaveLedgerMode(object? sender, EventArgs args)
    {
        if (_loading)
        {
            return;
        }

        _settings.SetLedgerMode(CurrentLedgerMode);
        _settingsStore.Save(_settings);
        UpdatePanelTitles();
    }

    private LedgerMode CurrentLedgerMode
    {
        get
        {
            if (_supplierRadio.Checked)
            {
                return LedgerMode.Supplier;
            }

            if (_customerRadio.Checked)
            {
                return LedgerMode.Customer;
            }

            return LedgerMode.Auto;
        }
    }

    private OutputFormats CurrentFormats => (_excelCheckBox.Checked ? OutputFormats.Excel : OutputFormats.None)
        | (_pdfCheckBox.Checked ? OutputFormats.Pdf : OutputFormats.None);

    private void UpdatePanelTitles()
    {
        var ledgerMode = CurrentLedgerMode;
        _titles[TreatmentRole.First].Text = ledgerMode switch
        {
            LedgerMode.Customer => "Lister les encaissements\nsans facture de ventes",
            LedgerMode.Supplier => "Lister les paiements\nsans facture",
            _ => "Lister paiements / encaissements\nsans facture"
        };
        _titles[TreatmentRole.Second].Text = ledgerMode switch
        {
            LedgerMode.Customer => "Lister les factures de ventes\nsans paiement",
            LedgerMode.Supplier => "Lister les factures\nsans paiements",
            _ => "Lister les factures\nsans paiements"
        };
        _titles[TreatmentRole.Both].Text = "Générer les 2 fichiers";
    }

    private void DropPanel_DragEnter(object? sender, DragEventArgs args)
    {
        args.Effect = args.Data?.GetDataPresent(DataFormats.FileDrop) == true
            ? DragDropEffects.Copy
            : DragDropEffects.None;
    }

    private async void DropPanel_DragDrop(TreatmentRole role, DragEventArgs args)
    {
        if (args.Data?.GetData(DataFormats.FileDrop) is string[] files)
        {
            await ProcessPathsAsync(files, role);
        }
    }

    private async void SelectCsv(TreatmentRole role)
    {
        using var dialog = new OpenFileDialog
        {
            Filter = "Exports CSV (*.csv)|*.csv|Tous les fichiers (*.*)|*.*",
            Multiselect = true,
            Title = "Choisir un grand livre CSV"
        };

        if (dialog.ShowDialog(this) == DialogResult.OK)
        {
            await ProcessPathsAsync(dialog.FileNames, role);
        }
    }

    private async Task ProcessPathsAsync(IReadOnlyList<string> paths, TreatmentRole role)
    {
        if (CurrentFormats == OutputFormats.None)
        {
            SetStatus("Format manquant", "Sélectionnez au moins un format.", isError: true);
            return;
        }

        SetBusy(true);
        SetStatus("Traitement", "Traitement en cours...", isError: false);

        try
        {
            var options = new ProcessingOptions(CurrentFormats, CurrentLedgerMode, role, PromptForMetadata);
            var result = await Task.Run(() => _processor.ProcessFiles(paths, options));
            SetStatus("Résultat", LedgerProcessor.BuildSummary(result), isError: false);
        }
        catch (Exception exception)
        {
            SetStatus("Erreur", exception.Message, isError: true);
        }
        finally
        {
            SetBusy(false);
        }
    }

    private FileMetadata? PromptForMetadata(string path)
    {
        if (InvokeRequired)
        {
            return (FileMetadata?)Invoke(new Func<FileMetadata?>(() => PromptForMetadata(path)));
        }

        return MetadataDialog.Show(this, path);
    }

    private void SetBusy(bool busy)
    {
        foreach (var panel in _dropPanels)
        {
            panel.Enabled = !busy;
        }

        _excelCheckBox.Enabled = !busy;
        _pdfCheckBox.Enabled = !busy;
        _autoRadio.Enabled = !busy;
        _supplierRadio.Enabled = !busy;
        _customerRadio.Enabled = !busy;
    }

    private void SetStatus(string title, string text, bool isError)
    {
        _statusTitle.Text = title;
        _statusText.Text = text;
        _statusTitle.ForeColor = isError ? Color.FromArgb(185, 28, 28) : Color.FromArgb(15, 23, 42);
        _statusText.ForeColor = isError ? Color.FromArgb(153, 27, 27) : Color.FromArgb(71, 85, 105);
    }
}

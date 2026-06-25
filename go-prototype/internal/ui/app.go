package ui

import (
	_ "embed"
	"errors"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/kamel934/RelanceTiime/go-prototype/internal/core"
	"github.com/kamel934/RelanceTiime/go-prototype/internal/winutil"
)

//go:embed assets/logo.png
var logoBytes []byte

type Application struct {
	app       fyne.App
	window    fyne.Window
	store     *core.SettingsStore
	settings  core.UserSettings
	processor *core.Processor

	excelCheck *widget.Check
	pdfCheck   *widget.Check
	modeRadio  *widget.RadioGroup
	firstCard  *DropCard
	secondCard *DropCard
	bothCard   *DropCard
	statusHead *widget.Label
	statusText *widget.Label

	loading bool
	busy    bool
	mutex   sync.Mutex
}

func NewApplication() *Application {
	fyneApp := app.NewWithID("fr.relancetiime.go-preview")
	fyneApp.Settings().SetTheme(newAppTheme())
	icon := fyne.NewStaticResource("logo.png", logoBytes)
	fyneApp.SetIcon(icon)

	window := fyneApp.NewWindow("Mise en forme - Grand livre (Go)")
	window.SetIcon(icon)
	window.Resize(fyne.NewSize(920, 740))
	window.CenterOnScreen()
	window.SetMaster()

	store := core.NewSettingsStore("")
	application := &Application{
		app:       fyneApp,
		window:    window,
		store:     store,
		settings:  store.Load(core.OutputExcel, core.LedgerAuto, false),
		processor: core.NewProcessor(core.NewExcelExporter(), core.NewPDFExporter()),
		loading:   true,
	}
	window.SetContent(application.buildContent())
	application.applySettings()
	application.loading = false
	application.updateCardTitles()
	window.SetOnDropped(application.onDropped)
	return application
}

func (a *Application) Run() {
	a.window.ShowAndRun()
}

func (a *Application) buildContent() fyne.CanvasObject {
	title := widget.NewLabel("Mise en forme du grand livre")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Importance = widget.HighImportance
	subtitle := widget.NewLabel("Déposez un export CSV pour générer les listes prêtes à envoyer.")
	header := container.NewVBox(title, subtitle)

	formatLabel := boldLabel("Format")
	a.excelCheck = widget.NewCheck("Excel", func(bool) { a.saveFormats() })
	a.pdfCheck = widget.NewCheck("PDF", func(bool) { a.saveFormats() })
	modeLabel := boldLabel("Grand livre")
	a.modeRadio = widget.NewRadioGroup([]string{"Auto", "Fournisseur", "Client"}, func(string) {
		a.saveLedgerMode()
	})
	a.modeRadio.Horizontal = true
	a.modeRadio.Required = true
	options := container.NewHBox(
		formatLabel,
		a.excelCheck,
		a.pdfCheck,
		layout.NewSpacer(),
		modeLabel,
		a.modeRadio,
	)

	a.firstCard = NewDropCard(
		"Lister paiements / encaissements\nsans facture",
		color.NRGBA{R: 31, G: 78, B: 120, A: 255},
		fyne.NewSize(370, 170),
		func() { a.selectFiles(core.RoleFirst) },
	)
	a.secondCard = NewDropCard(
		"Lister les factures\nsans paiements",
		color.NRGBA{R: 138, G: 106, B: 19, A: 255},
		fyne.NewSize(370, 170),
		func() { a.selectFiles(core.RoleSecond) },
	)
	a.bothCard = NewDropCard(
		"Générer les 2 fichiers",
		color.NRGBA{R: 51, G: 65, B: 85, A: 255},
		fyne.NewSize(460, 110),
		func() { a.selectFiles(core.RoleBoth) },
	)
	topCards := container.New(&topCardsLayout{}, a.firstCard, a.secondCard)
	bottomCard := container.New(&bottomCardLayout{}, a.bothCard)

	note := widget.NewLabel("Le dépôt direct d'un CSV sur l'icône du .exe génère la première liste du type détecté.")
	note.Alignment = fyne.TextAlignCenter
	note.Importance = widget.LowImportance

	a.statusHead = boldLabel("Résultat")
	a.statusText = widget.NewLabel("Déposez un CSV pour lancer un traitement.")
	a.statusText.Wrapping = fyne.TextWrapWord
	statusScroll := container.NewVScroll(a.statusText)
	statusBody := container.NewBorder(a.statusHead, nil, nil, nil, statusScroll)
	statusBody = container.NewPadded(statusBody)
	statusOutline := canvas.NewRectangle(color.White)
	statusOutline.StrokeColor = color.NRGBA{R: 203, G: 213, B: 225, A: 255}
	statusOutline.StrokeWidth = 1
	statusPanel := container.NewStack(statusOutline, statusBody)

	return container.New(
		&mainLayout{},
		header,
		options,
		topCards,
		bottomCard,
		note,
		statusPanel,
	)
}

func boldLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Bold: true}
	return label
}

func (a *Application) applySettings() {
	a.excelCheck.SetChecked(a.settings.Excel)
	a.pdfCheck.SetChecked(a.settings.PDF)
	switch a.settings.ParsedLedgerMode() {
	case core.LedgerModeSupplier:
		a.modeRadio.SetSelected("Fournisseur")
	case core.LedgerModeCustomer:
		a.modeRadio.SetSelected("Client")
	default:
		a.modeRadio.SetSelected("Auto")
	}
}

func (a *Application) saveFormats() {
	if a.loading {
		return
	}
	a.settings.Excel = a.excelCheck.Checked
	a.settings.PDF = a.pdfCheck.Checked
	if err := a.store.Save(a.settings); err != nil {
		a.setStatus("Erreur", err.Error(), true)
	}
}

func (a *Application) saveLedgerMode() {
	if a.loading {
		return
	}
	a.settings.SetLedgerMode(a.currentLedgerMode())
	if err := a.store.Save(a.settings); err != nil {
		a.setStatus("Erreur", err.Error(), true)
	}
	a.updateCardTitles()
}

func (a *Application) currentLedgerMode() core.LedgerMode {
	switch a.modeRadio.Selected {
	case "Fournisseur":
		return core.LedgerModeSupplier
	case "Client":
		return core.LedgerModeCustomer
	default:
		return core.LedgerAuto
	}
}

func (a *Application) currentFormats() core.OutputFormats {
	formats := core.OutputNone
	if a.excelCheck.Checked {
		formats |= core.OutputExcel
	}
	if a.pdfCheck.Checked {
		formats |= core.OutputPDF
	}
	return formats
}

func (a *Application) updateCardTitles() {
	switch a.currentLedgerMode() {
	case core.LedgerModeSupplier:
		a.firstCard.SetTitle("Lister les paiements\nsans facture")
		a.secondCard.SetTitle("Lister les factures\nsans paiements")
	case core.LedgerModeCustomer:
		a.firstCard.SetTitle("Lister les encaissements\nsans facture de ventes")
		a.secondCard.SetTitle("Lister les factures de ventes\nsans paiement")
	default:
		a.firstCard.SetTitle("Lister paiements / encaissements\nsans facture")
		a.secondCard.SetTitle("Lister les factures\nsans paiements")
	}
	a.bothCard.SetTitle("Générer les 2 fichiers")
}

func (a *Application) selectFiles(role core.TreatmentRole) {
	if a.busy {
		return
	}
	paths, err := winutil.SelectCSVFiles()
	if err != nil {
		a.setStatus("Erreur", err.Error(), true)
		return
	}
	if len(paths) > 0 {
		a.processPaths(paths, role)
	}
}

func (a *Application) onDropped(position fyne.Position, uris []fyne.URI) {
	if a.busy {
		return
	}
	role, found := a.roleAt(position)
	if !found {
		a.setStatus("Zone de dépôt", "Déposez le CSV dans l'un des trois encadrés.", true)
		return
	}
	paths := make([]string, 0, len(uris))
	for _, uri := range uris {
		if uri.Scheme() == "file" {
			paths = append(paths, filepath.FromSlash(uri.Path()))
		}
	}
	if len(paths) > 0 {
		a.processPaths(paths, role)
	}
}

func (a *Application) roleAt(position fyne.Position) (core.TreatmentRole, bool) {
	for _, candidate := range []struct {
		card *DropCard
		role core.TreatmentRole
	}{
		{a.firstCard, core.RoleFirst},
		{a.secondCard, core.RoleSecond},
		{a.bothCard, core.RoleBoth},
	} {
		absolute := a.app.Driver().AbsolutePositionForObject(candidate.card)
		size := candidate.card.Size()
		if position.X >= absolute.X && position.X <= absolute.X+size.Width &&
			position.Y >= absolute.Y && position.Y <= absolute.Y+size.Height {
			return candidate.role, true
		}
	}
	return 0, false
}

func (a *Application) processPaths(paths []string, role core.TreatmentRole) {
	formats := a.currentFormats()
	if formats == core.OutputNone {
		a.setStatus("Format manquant", "Sélectionnez au moins un format.", true)
		return
	}

	mode := a.currentLedgerMode()
	a.setBusy(true)
	a.setStatus("Traitement", "Traitement en cours...", false)

	go func() {
		result, err := a.processor.ProcessFiles(paths, core.ProcessingOptions{
			Formats:          formats,
			LedgerMode:       mode,
			Role:             role,
			MetadataProvider: a.promptForMetadata,
		})
		fyne.Do(func() {
			defer a.setBusy(false)
			if err != nil {
				a.setStatus("Erreur", err.Error(), true)
				return
			}
			a.setStatus("Résultat", core.BuildSummary(result), false)
		})
	}()
}

func (a *Application) promptForMetadata(path string) (*core.FileMetadata, error) {
	type response struct {
		metadata *core.FileMetadata
		err      error
	}
	channel := make(chan response, 1)
	fyne.Do(func() {
		client := widget.NewEntry()
		year := widget.NewEntry()
		year.SetText(fmt.Sprintf("%d", time.Now().Year()))
		form := dialog.NewForm(
			"Nom de fichier non standard",
			"OK",
			"Annuler",
			[]*widget.FormItem{
				widget.NewFormItem("Client", client),
				widget.NewFormItem("Année", year),
			},
			func(confirmed bool) {
				if !confirmed {
					channel <- response{err: errors.New("traitement annulé")}
					return
				}
				clientValue := strings.ToUpper(strings.TrimSpace(client.Text))
				yearValue := strings.TrimSpace(year.Text)
				if clientValue == "" || yearValue == "" {
					channel <- response{err: errors.New("le client et l'année sont requis")}
					return
				}
				channel <- response{metadata: &core.FileMetadata{Client: clientValue, Year: yearValue}}
			},
			a.window,
		)
		form.Resize(fyne.NewSize(430, 220))
		form.Show()
	})
	answer := <-channel
	return answer.metadata, answer.err
}

func (a *Application) setBusy(busy bool) {
	a.mutex.Lock()
	a.busy = busy
	a.mutex.Unlock()

	a.firstCard.SetEnabled(!busy)
	a.secondCard.SetEnabled(!busy)
	a.bothCard.SetEnabled(!busy)
	if busy {
		a.excelCheck.Disable()
		a.pdfCheck.Disable()
		a.modeRadio.Disable()
	} else {
		a.excelCheck.Enable()
		a.pdfCheck.Enable()
		a.modeRadio.Enable()
	}
}

func (a *Application) setStatus(title, text string, isError bool) {
	a.statusHead.SetText(title)
	a.statusText.SetText(text)
	if isError {
		a.statusHead.Importance = widget.DangerImportance
	} else {
		a.statusHead.Importance = widget.MediumImportance
	}
	a.statusHead.Refresh()
}

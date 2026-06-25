package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	fyneTest "fyne.io/fyne/v2/test"

	"github.com/kamel934/RelanceTiime/go-prototype/internal/core"
)

func TestDropCardRendererKeepsTwoLineTitlesVisible(t *testing.T) {
	card := NewDropCard(
		"Lister les encaissements\nsans facture de ventes",
		color.NRGBA{R: 31, G: 78, B: 120, A: 255},
		fyne.NewSize(370, 170),
		nil,
	)
	renderer := card.CreateRenderer().(*dropCardRenderer)
	renderer.Layout(fyne.NewSize(420, 190))
	renderer.Refresh()

	if renderer.titleOne.Text != "Lister les encaissements" ||
		renderer.titleTwo.Text != "sans facture de ventes" {
		t.Fatalf("titre incorrect : %q / %q", renderer.titleOne.Text, renderer.titleTwo.Text)
	}
	if renderer.titleTwo.Position().Y+renderer.titleTwo.Size().Height >= renderer.button.Position().Y {
		t.Fatal("le titre chevauche le bouton")
	}
}

func TestRoleAtRoutesEachDropZone(t *testing.T) {
	testApp := fyneTest.NewApp()
	defer testApp.Quit()
	window := testApp.NewWindow("test")

	application := &Application{app: testApp, window: window}
	application.firstCard = NewDropCard("Premier", color.Black, fyne.NewSize(370, 170), nil)
	application.secondCard = NewDropCard("Second", color.Black, fyne.NewSize(370, 170), nil)
	application.bothCard = NewDropCard("Deux", color.Black, fyne.NewSize(460, 110), nil)
	topCards := fyne.NewContainerWithLayout(&topCardsLayout{}, application.firstCard, application.secondCard)
	bottomCard := fyne.NewContainerWithLayout(&bottomCardLayout{}, application.bothCard)
	content := fyne.NewContainerWithLayout(
		&mainLayout{},
		fyne.NewContainerWithoutLayout(),
		fyne.NewContainerWithoutLayout(),
		topCards,
		bottomCard,
		fyne.NewContainerWithoutLayout(),
		fyne.NewContainerWithoutLayout(),
	)
	window.SetContent(content)
	window.Resize(fyne.NewSize(920, 740))
	window.Show()

	tests := []struct {
		card *DropCard
		role core.TreatmentRole
	}{
		{application.firstCard, core.RoleFirst},
		{application.secondCard, core.RoleSecond},
		{application.bothCard, core.RoleBoth},
	}
	for _, test := range tests {
		absolute := testApp.Driver().AbsolutePositionForObject(test.card)
		position := fyne.NewPos(
			absolute.X+test.card.Size().Width/2,
			absolute.Y+test.card.Size().Height/2,
		)
		role, found := application.roleAt(position)
		if !found || role != test.role {
			t.Fatalf("zone mal routée : attendu %d, obtenu %d (trouvé=%v)", test.role, role, found)
		}
	}
}

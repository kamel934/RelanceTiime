# Liste Tiime Go Preview

Prototype parallèle de Liste Tiime, écrit en Go avec Fyne. La version C#
WinForms reste la version officielle tant que cette variante n'offre pas une
parité complète et un gain mesurable.

## Développement

Prérequis Windows :

- Go 1.26.4 ;
- MSYS2 avec `mingw-w64-ucrt-x86_64-toolchain` ;
- `C:\msys64\ucrt64\bin` dans le `PATH`.

```powershell
go test ./...
.\scripts\build-go.ps1
```

Le binaire est créé dans `publish\Liste Tiime Go.exe`.

## Bibliothèques principales

- Fyne 2.7.4 pour l'interface ;
- Excelize 2.10.1 pour Excel ;
- Maroto 2.4.0 pour PDF ;
- shopspring/decimal pour les montants.

Le prototype lit et écrit le même fichier de réglages que la version C# :
`%APPDATA%\RelanceTiime\settings.json`.

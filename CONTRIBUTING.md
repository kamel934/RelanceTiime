# Contribuer

Merci de contribuer à Liste Tiime.

## Règles

- Ne jamais committer de CSV, Excel, PDF ou données client réelles.
- Utiliser uniquement des données synthétiques dans les tests.
- Conserver la compatibilité Windows 10/11 64 bits et .NET 10.
- Ajouter ou adapter les tests pour chaque changement métier.
- Vérifier que les fichiers Excel s'ouvrent sans réparation et que les PDF
  commencent par `%PDF-`.

## Vérifications locales

```powershell
dotnet restore ListeTiime.slnx
dotnet test ListeTiime.slnx -c Release
dotnet publish src\ListeTiime.App\ListeTiime.App.csproj -c Release -r win-x64 -o publish
```

Les pull requests doivent expliquer le comportement modifié et les tests
effectués.

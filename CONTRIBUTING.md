# Contribuer

Merci de contribuer à Liste Tiime.

## Règles

- Ne jamais committer de CSV, Excel, PDF ou données client réelles.
- Utiliser uniquement des données synthétiques dans les tests.
- Conserver la compatibilité Windows et Python 3.12.
- Ajouter ou adapter les tests pour chaque changement métier.
- Vérifier que les fichiers Excel s'ouvrent sans réparation et que les PDF
  commencent par `%PDF-`.

## Vérifications locales

```powershell
python -m venv .venv
.\.venv\Scripts\python.exe -m pip install -r requirements-dev.txt
.\.venv\Scripts\python.exe -m pytest -q
.\.venv\Scripts\pyinstaller.exe --clean --noconfirm "Liste Tiime.spec"
```

Les pull requests doivent expliquer le comportement modifié et les tests
effectués.

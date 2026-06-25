# Changelog

## 2.0.0 - 2026-06-25

- Réécriture complète en C# WinForms sous .NET 10.
- Génération Excel via ClosedXML et PDF via MigraDoc/PDFsharp, sans Microsoft Excel.
- Démarrage plus rapide et interface non bloquante pendant les traitements.
- Tests xUnit pour les traitements fournisseurs/clients, colonnes vides, avoirs, remboursements, PDF et réglages.
- Suppression de l'ancien code Python/PyInstaller et de la documentation SignPath refusée.

## 1.0.0 - 2026-06-24

- Détection automatique des grands livres fournisseurs et clients.
- Génération des listes fournisseurs et clients en Excel et PDF.
- Choix persistant du format et du type de grand livre.
- Glisser-déposer dans l'interface et directement sur l'exécutable.
- Pipeline reproductible de build et release.

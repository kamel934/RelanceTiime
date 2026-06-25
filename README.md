# Liste Tiime

![Logo Liste Tiime](logo_compta_premium.png)

Liste Tiime transforme localement un export CSV de grand livre en fichiers
Excel et PDF prêts à envoyer au client. L'application est réécrite en C#
WinForms/.NET 10 pour démarrer plus vite et ne dépend plus de Python,
PyInstaller ou Microsoft Excel.

## Fonctionnalités

- Détection automatique d'un grand livre fournisseur (`401`) ou client (`411`).
- Paiements fournisseurs sans facture.
- Factures fournisseurs sans paiement.
- Encaissements clients sans facture de vente.
- Factures de vente sans paiement.
- Export Excel, PDF ou les deux.
- Glisser-déposer dans l'application ou directement sur l'exécutable.
- Masquage automatique des colonnes de remboursements ou d'avoirs vides.
- Génération PDF directe en A4 portrait, avec en-tête répété et pagination.

## Confidentialité

Le traitement est entièrement local. Les CSV et les documents générés ne sont
envoyés à aucun service distant. Les workflows GitHub utilisent uniquement des
données de test synthétiques et ne contiennent aucune donnée client.

## Télécharger

Téléchargez `Liste Tiime.exe` depuis la page
[Releases](https://github.com/kamel934/RelanceTiime/releases).

Le binaire publié est autonome pour Windows 10/11 64 bits. Il n'est pas signé
avec un certificat public, donc Windows peut encore afficher un avertissement
SmartScreen sur un nouveau poste.

Vous pouvez vérifier l'intégrité du téléchargement avec le fichier
`SHA256SUMS.txt` publié dans la même release :

```powershell
Get-FileHash ".\Liste Tiime.exe" -Algorithm SHA256
```

## Utilisation

1. Ouvrez `Liste Tiime.exe`.
2. Choisissez `Excel`, `PDF` ou les deux.
3. Laissez le type sur `Auto`, ou forcez `Fournisseur` / `Client`.
4. Déposez le CSV dans la zone voulue.

Un CSV glissé directement sur l'icône génère la première liste correspondant
au type de grand livre détecté. Le succès est silencieux dans ce mode ; les
erreurs restent affichées clairement.

## Format CSV attendu

Le fichier doit être séparé par `;`, encodé en Windows-1252 et contenir au
minimum :

- `N° Compte`
- `Libellé du compte`
- `Date`
- `Journal`
- `N° de pièce`
- `Libellé mouvement`
- `Montant Débit`
- `Montant Crédit`

## Développement

Prérequis : Windows et .NET SDK 10.

```powershell
dotnet restore ListeTiime.slnx
dotnet test ListeTiime.slnx -c Release
dotnet publish src\ListeTiime.App\ListeTiime.App.csproj -c Release -r win-x64 -o publish
```

Le binaire autonome est créé dans `publish\Liste Tiime.exe`.

## Licence

Ce projet est distribué sous licence [MIT](LICENSE).

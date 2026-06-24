# Candidature SignPath Foundation

Formulaire : <https://signpath.org/apply>

## Informations du projet

- Nom : Liste Tiime
- Handle proposé : `liste-tiime`
- Mainteneur : `kamel934`
- Dépôt : <https://github.com/kamel934/RelanceTiime>
- Page de téléchargement :
  <https://github.com/kamel934/RelanceTiime/releases>
- Licence : MIT
- Plateforme : Windows
- Type d'artefact : exécutable PE `.exe`
- Nom de l'artefact : `Liste Tiime.exe`

## Description

Liste Tiime est un outil Windows open source qui transforme localement des
exports CSV de grands livres comptables en listes Excel et PDF. Il détecte les
grands livres fournisseurs et clients et génère les listes de paiements,
encaissements et factures sans correspondance.

Le traitement est entièrement local. Aucune donnée comptable n'est envoyée à
un service distant.

## Build et provenance

- GitHub Actions sur `windows-latest`
- Python 3.12
- PyInstaller 6.20.0
- Workflow :
  `.github/workflows/build-release.yml`
- Dépendances épinglées dans `requirements.txt` et `requirements-dev.txt`
- Tests synthétiques dans `tests/`
- Release initiale : `v1.0.0`

## Politique demandée

- Certificat : SignPath Foundation
- Approbation manuelle pour chaque release
- Produit imposé : `Liste Tiime`
- Version produit imposée : identique au tag de release
- Nom de fichier imposé : `Liste Tiime.exe`
- Build autorisé : uniquement le workflow GitHub Actions du dépôt
- Runners autorisés : uniquement les runners GitHub hébergés

## Configuration GitHub après approbation

Secret :

- `SIGNPATH_API_TOKEN`

Variables :

- `SIGNPATH_ORGANIZATION_ID`
- `SIGNPATH_PROJECT_SLUG`
- `SIGNPATH_SIGNING_POLICY_SLUG`
- `SIGNPATH_ARTIFACT_CONFIGURATION_SLUG`

Le workflow est déjà prêt à utiliser ces valeurs et à refuser une release si
la signature téléchargée n'est pas `Valid`.

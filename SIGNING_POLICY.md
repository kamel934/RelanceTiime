# Politique de signature

Free code signing provided by [SignPath.io](https://signpath.io/), certificate
by [SignPath Foundation](https://signpath.org/).

## Responsabilités

- Auteur : `kamel934`
- Reviewer : `kamel934`
- Approver : `kamel934`

## Provenance

- Les builds de release sont exécutés uniquement sur des runners GitHub
  hébergés.
- Le code source correspondant est identifié par un tag Git signé ou annoté.
- L'exécutable non signé est produit par GitHub Actions et téléversé comme
  artefact avant toute demande de signature.
- La demande SignPath est soumise par
  `signpath/github-action-submit-signing-request@v2`.
- Une release configurée avec SignPath n'est publiée que si
  `Get-AuthenticodeSignature` retourne `Valid`.
- Chaque release publie un fichier `SHA256SUMS.txt`.

## Configuration SignPath attendue

Secret GitHub :

- `SIGNPATH_API_TOKEN`

Variables GitHub :

- `SIGNPATH_ORGANIZATION_ID`
- `SIGNPATH_PROJECT_SLUG`
- `SIGNPATH_SIGNING_POLICY_SLUG`
- `SIGNPATH_ARTIFACT_CONFIGURATION_SLUG`

Tant que ces valeurs ne sont pas configurées, le workflow peut publier une
release explicitement marquée comme non signée afin de documenter le projet et
sa première version. Une nouvelle exécution du workflow remplacera ensuite
l'asset par l'exécutable signé.

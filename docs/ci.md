# CI und Dependabot

Zwei Dateien im Repository kümmern sich um den automatischen Betrieb: der Workflow `.github/workflows/ci.yml` prüft jeden Stand, und `.github/dependabot.yml` meldet neue Versionen von Abhängigkeiten.

## Der Workflow

Die Datei `.github/workflows/ci.yml` heisst `CI` und läuft bei zwei Ereignissen:

| Ereignis | Wann |
|---|---|
| `push` auf `master` | Nach jedem Push auf den Hauptbranch |
| `pull_request` | Für jeden Pull Request, unabhängig vom Zielbranch |

Die Berechtigungen sind auf `contents: read` beschränkt: Der Workflow liest das Repository, er schreibt nichts und braucht kein Geheimnis.

Läuft eine Prüfung zweimal für denselben Branch, bricht die ältere der beiden ab (`cancel-in-progress`). Das gilt nur für Pull Requests: Ein Lauf auf `master` läuft immer zu Ende und hinterlässt ein vollständiges Ergebnis.

### Die Aktionen sind festgepinnt

Die Aktionen stehen nicht auf einem Tag, sondern auf dem SHA eines Commits:

```yaml
uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
```

Ein Tag kann verschoben werden, ein Commit nicht. Der Kommentar hinter dem SHA nennt die Version, damit der Stand lesbar bleibt und eine Aktualisierung nachvollziehbar ist.

Der Token des Checkouts wird nicht in der Git-Konfiguration des Arbeitsverzeichnisses behalten (`persist-credentials: false`). Nach dem Auschecken wird er nicht mehr gebraucht, so bleibt er auch nicht auf der Maschine des Runners liegen.

### Der Job `go`

Der Job prüft den Code selbst. Die Go-Version kommt aus `go.mod`, es ist keine im Workflow eingetragene.

| Schritt | Befehl | Was er findet |
|---|---|---|
| Formatierung | `gofmt -l .` | Dateien, die nicht mit `gofmt` formatiert sind. Der Job bricht ab, wenn der Befehl eine Datei nennt. |
| Prüfung | `go vet ./...` | Verdächtige Konstrukte, zum Beispiel eine unbenutzte Variable oder ein falsch geformtes `Printf`. |
| Build | `CGO_ENABLED=0 go build ./...` | Ob der Code ohne CGO baut. Das ist die Bedingung für das statische Binary im Container. |
| Tests | `go test ./... -race -count=1` | Ob die Tests grün sind. `-race` sucht nebenläufige Fehler, `-count=1` umgeht den Test-Cache. |

### Der Job `vulncheck`

Der Job sucht bekannte Schwachstellen im Code:

```bash
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
govulncheck ./...
```

Die Version des Werkzeugs steht im Workflow (`v1.9.0`) und nicht über `latest`: `latest` würde die Prüfung von Lauf zu Lauf ändern.

`govulncheck` braucht Netzzugriff auf die Schwachstellendatenbank von Go. Ohne Netz schlägt der Job fehl, das ist kein Fehler des Projekts.

Das Werkzeug meldet die Schwachstellen, die der Code tatsächlich erreichen kann, dazu diejenigen der Standardbibliothek. Auf dem aktuellen Stand wurde lokal nichts gefunden.

### Der Job `docs`

Der Job baut die Dokumentation mit Python 3.12 und dem strikten Modus. Er hat drei Schritte:

```bash
pip install --dry-run -r requirements-docs.txt
pip install -r requirements-docs.txt
mkdocs build --strict
```

Der erste Schritt heisst `Check documentation requirements` und installiert nichts: Er löst die Anforderungen nur auf. So werden Konflikte der Versionsbereiche früh sichtbar, noch bevor etwas installiert ist. Schlägt die Auflösung fehl, meldet der Schritt den Fehler und bricht den Job ab.

`--strict` macht aus jeder Warnung einen Fehler. Ein Link auf eine nicht vorhandene Seite, eine Datei ohne Eintrag in der Navigation oder ein unvollständiger Verweis bricht den Job ab.

### Der Job `docker`

Der Job baut das Image:

```bash
docker build .
```

Er baut nur. Er schiebt nichts in eine Registry und braucht dafür auch keine Zugangsdaten. Das fertige Image entsteht auf dem Runner und ist danach wieder weg.

### Wie die Prüfungen lokal laufen

Alle vier Jobs lassen sich auf dem Rechner nachstellen:

```bash
# Job go
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "These files are not formatted with gofmt:"
  echo "$unformatted"
  exit 1
fi
go vet ./...
CGO_ENABLED=0 go build ./...
go test ./... -race -count=1

# Job vulncheck (braucht Netzzugriff auf die Schwachstellendatenbank)
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
"$(go env GOPATH)/bin/govulncheck" ./...

# Job docs
python3 -m venv .venv-docs
. .venv-docs/bin/activate
pip install --dry-run -r requirements-docs.txt
pip install -r requirements-docs.txt
mkdocs build --strict

# Job docker
docker build .
```

Der Doku-Build landet im Ordner `site/`, der nicht im Repository liegt.

## Dependabot

Die Datei `.github/dependabot.yml` meldet neue Versionen. Sie kennt vier Ökosysteme:

| Ökosystem | Was davon betroffen ist | Präfix der Commits |
|---|---|---|
| `github-actions` | Die Aktionen im Workflow, etwa `actions/checkout` | `ci(deps)` |
| `gomod` | Die Module in `go.mod` und `go.sum` | `build(deps)` |
| `docker` | Die Basis-Images im `Dockerfile` | `build(deps)` |
| `pip` | Die Werkzeuge in `requirements-docs.txt` | `build(deps)` |

Für alle vier gilt derselbe Rhythmus:

| Angabe | Wert |
|---|---|
| Intervall | wöchentlich |
| Tag | Montag |
| Zeit | 06:00 |
| Zeitzone | `Europe/Zurich` |
| Offene Pull Requests | höchstens 5 je Ökosystem |

Die Updates werden gruppiert. Bei drei Gruppen kommen Minor- und Patch-Updates zusammen in einem Pull Request, ein Major-Update kommt in einem eigenen:

| Gruppe | Muster | Update-Arten |
|---|---|---|
| `actions` | alle | minor, patch |
| `go-modules` | alle | minor, patch |
| `docker-images` | alle ausser `golang` | minor, patch |
| `docs-tools` | alle | alle |

Die Gruppe `docs-tools` ist die einzige ohne Einschränkung: Sie fasst alle Update-Arten in einem Pull Request zusammen. Der Grund ist die Kopplung der Werkzeuge. Eine neue Version des einen Werkzeugs kann eine Version des anderen verlangen, die der Bereich in `requirements-docs.txt` nicht zulässt. Getrennte Updates liessen sich dann nicht auflösen, die Anforderungsdatei ändert sich deshalb als Ganzes.

Die Pull Requests tragen das Label `dependencies`, die der GitHub-Actions zusätzlich `github_actions`.

### Die Update-Policy für Go

Die Go-Version wird von Dependabot nur eingeschränkt aktualisiert:

| Änderung | Wie sie erfolgt |
|---|---|
| Patch des Builder-Images (1.27.3 zu 1.27.4) | Automatisch, als eigener Pull Request |
| Minor oder Major der Go-Version (1.27 zu 1.28) | Von Hand, in einem Commit, der `go.mod` und das `Dockerfile` zusammen ändert |

Der Grund für die Trennung:

- Die Go-Standardbibliothek steckt statisch im ausgelieferten Binary. Patch-Releases bringen fast immer Korrekturen für Sicherheitslücken in `net/http` und `crypto/tls` mit. Diese Fixes sollen ohne Warten auf Menschen ankommen.
- Die `go`-Direktive in `go.mod` nennt nur die Mindestversion. Ein Patch des Builder-Images baut darum weiterhin gültig, ein Minor-Wechsel nicht: Er verlangt eine neue `go`-Direktive in `go.mod`.
- Ein Minor-Wechsel ändert darum immer zwei Dateien. Das macht der Automatismus nicht, das bleibt eine bewusste Handarbeit mit Review.

Der Gruppe `docker-images` ist das Go-Image deshalb ausdrücklich ausgenommen: Ein Patch der Standardbibliothek soll nicht an einem anderen Image-Update hängen und nicht auf dessen Freigabe warten.

Die `FROM`-Zeile im `Dockerfile` nennt Tag und Digest, das Basis-Image ist damit an einen festen Stand gebunden. Ob die `ignore`-Regel für `golang` mit dieser Form greift, ist erst nach dem ersten Lauf auf GitHub belegt: Die Regel nennt den Namen der Abhängigkeit, und der Name bleibt auch in der gepinnten Zeile `golang`.

### Ein Hinweis zur Toolchain

Steigt die `go`-Direktive in `go.mod` über die Go-Version des Builder-Images, braucht der Build eine neuere Toolchain. Go lädt sie nach, solange `GOTOOLCHAIN` auf dem Standard `auto` steht, und bricht stattdessen ab, wenn `GOTOOLCHAIN=local` gesetzt ist. Der Job `docker` in der CI zeigt das früh, weil er das Image mit dem Builder baut.

Der Fall tritt nur auf, wenn eine Aktualisierung eines Moduls eine neuere Go-Version verlangt. Dann gehört die `go`-Direktive in `go.mod` und das Tag im `Dockerfile` in denselben Commit.

## Was noch aussteht

- Der erste Lauf des Workflows erfolgt mit dem Pull Request des Branches `feat/phase-3-betrieb`: Der Workflow startet bei einem `push` auf `master` und bei jedem `pull_request`, ein Push nur auf einen Feature-Branch löst ihn nicht aus. Die Dependabot-Konfiguration wird erst mit dem Merge auf `master` wirksam.
- Ein Update, das die Go-Version anhebt, braucht zusätzlich zum `Dockerfile` eine neue `go`-Direktive in `go.mod`.
- Ob die `ignore`-Regel für `golang` die gepinnte `FROM`-Zeile mit Tag und Digest trifft und ob Dependabot die Versionskommentare hinter den SHAs pflegt, zeigt der erste Lauf nach dem Push.
- Dass die Gruppe `docs-tools` ohne Einschränkung alle Update-Arten in einem Pull Request zusammenfasst, ist erst nach dem Merge belegt. Dass die Werkzeuge in `requirements-docs.txt` zusammen auflösbar sind, ist geprüft; wie Dependabot sie gruppiert, noch nicht.

## Weiterführend

- [Beitragen](contributing.md): Branch, Prüfungen und Sprachregeln
- [Betrieb](betrieb.md): der Container, der aus demselben `Dockerfile` gebaut wird
- [Architektur](architecture.md): die Pakete, die der Job `go` prüft

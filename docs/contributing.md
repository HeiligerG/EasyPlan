# Beitragen

## Arbeitsweise

- Ein Feature-Branch pro Phase: `feat/phase-1-kern`, `feat/phase-2-ui`, `feat/phase-3-betrieb`. Nie direkt auf dem Hauptbranch `master`.
- Jede Phase durchläuft dieselben Schritte: Umsetzung, Tests, Review, Doku.
- Zusammengeführt wird über einen Pull Request auf `master`, nicht über einen lokalen Merge.
- Commit, Push und Pull Request macht der planner, und erst nach der Bestätigung durch den Menschen.
- `Konzept.md` ist das ursprüngliche Konzept und gibt den Rahmen vor. Verbindlich für die Umsetzung sind der Code und die Doku unter `docs/`. Abweichungen werden gemeldet und in der Doku festgehalten, nicht im Code stillschweigend aufgelöst.

## Prüfungen

Vor jedem Stand:

```bash
go vet ./...
go test ./...
```

Für die Dokumentation:

```bash
mkdocs build --strict
```

Die Werkzeuge dafür stehen in `requirements-docs.txt`:

```bash
python3 -m venv .venv-docs
. .venv-docs/bin/activate
pip install -r requirements-docs.txt
```

Der Build landet im Ordner `site/`, der nicht im Repository liegt.

## Teststrategie

Die Suite umfasst über 280 Tests. Sie prüft die Bausteine auf die Weise, die zu ihnen passt:

- `internal/plan` ist rein. Ein Test braucht darum kein Datum aus der Uhr, er übergibt eines. Die Berechnung wird mit festen Eingaben geprüft, die mit dem Konzeptbeispiel übereinstimmen.
- `internal/store` läuft gegen eine SQLite-Datenbank im Arbeitsspeicher (`store.OpenMemory()`). Die Migrationen, die Pragmas und die Fehlerfälle des Stores stehen dort.
- `internal/web` prüft die Handler mit `httptest` gegen dieselbe Datenbank im Arbeitsspeicher. Die Uhr ist eine injizierte Funktion, ein Test gibt darum ein festes Datum vor und prüft so den Stichtag, den Monatswechsel und den Sommerzeitwechsel.
- Die reinen Funktionen der Web-Schicht (`BuildSummary`, `ValidateEntryForm`, `ValidateAccountName`, `AccountRows`, `BuildTimeline`, `ParseAccountFilter`, `ParseAmount`) werden direkt geprüft, ohne Server und ohne Datenbank.
- Der Start des Servers läuft in einem eigenen Prozess, damit das Laden der Zeitzone, das Setzen von `time.Local` und das saubere Beenden geprüft sind.

Gelegentlich wird zusätzlich eine Mutationsprobe gefahren: Ein paar Zeilen des Produktivcodes werden absichtlich verändert, und die Tests müssen rot werden. Sie deckt ab, dass die Tests wirklich greifen, und nicht nur grün sind.

## Doku

Die Doku liegt unter `docs/` und wird mit MkDocs gebaut.

- Jede neue Seite gehört in die Navigation von `mkdocs.yml`, sonst bleibt sie verwaist.
- Architekturentscheidungen kommen als ADR unter `docs/adr/` mit Nummer, Status und Datum.
- Was sich geändert hat, kommt unter `## [Unreleased]` in `docs/changelog.md`, formuliert aus Sicht der Nutzerin.
- Code-Beispiele in der Doku müssen zum aktuellen Code passen. Im Zweifel den Code lesen statt raten.

## Sprache

- Doku, Pläne, Berichte und Kommentare in der Dokumentation: Deutsch mit Schweizer Schreibweise, ohne ß.
- Code, Bezeichner, Commit-Messages und Code-Kommentare: Englisch.

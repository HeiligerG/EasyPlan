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

## Doku

Die Doku liegt unter `docs/` und wird mit MkDocs gebaut.

- Jede neue Seite gehört in die Navigation von `mkdocs.yml`, sonst bleibt sie verwaist.
- Architekturentscheidungen kommen als ADR unter `docs/adr/` mit Nummer, Status und Datum.
- Was sich geändert hat, kommt unter `## [Unreleased]` in `docs/changelog.md`, formuliert aus Sicht der Nutzerin.
- Code-Beispiele in der Doku müssen zum aktuellen Code passen. Im Zweifel den Code lesen statt raten.

## Sprache

- Doku, Pläne, Berichte und Kommentare in der Dokumentation: Deutsch mit Schweizer Schreibweise, ohne ß.
- Code, Bezeichner, Commit-Messages und Code-Kommentare: Englisch.

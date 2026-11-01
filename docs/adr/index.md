# Architekturentscheidungen

Ein ADR hält eine Entscheidung fest, die das Projekt prägt: Kontext, Entscheidung und Folgen. AdRs werden nummeriert und nie überschrieben. Eine spätere Entscheidung, die eine frühere ersetzt, bekommt eine eigene Nummer und verweist in der alten auf den Nachfolger.

| ADR | Titel | Status | Datum |
|---|---|---|---|
| [ADR-0001](0001-sqlite-ohne-cgo.md) | Eine SQLite-Datei ohne CGO (`modernc.org/sqlite`) | accepted | 2026-10-30 |
| [ADR-0002](0002-rappen-und-rundung.md) | Beträge als Rappen (`int64`), Rundung nur bei der Anzeige | accepted | 2026-10-30 |
| [ADR-0003](0003-kalendertage-utc.md) | Kalendertage als UTC-Mitternacht, Termine nur aus dem Startdatum | accepted | 2026-10-30 |
| [ADR-0004](0004-naechste-zahlung-pro-konto.md) | Nächste Zahlung pro Konto | accepted | 2026-10-30 |
| [ADR-0005](0005-origin-pruefung-statt-csrf-token.md) | Origin-Prüfung statt CSRF-Token | accepted | 2026-10-31 |
| [ADR-0006](0006-timeline-zeitraum-und-monatsgruppen.md) | Zeitraum und Monatsgruppen der Timeline | accepted | 2026-10-31 |
| [ADR-0007](0007-backup-als-post.md) | Backup als POST mit temporärer Datei im Datenordner | accepted | 2026-10-31 |
| [ADR-0008](0008-bind-default-und-image-override.md) | Bind-Default `127.0.0.1:8080` und Image-Override `:8080` | accepted | 2026-10-31 |
| [ADR-0009](0009-dependabot-update-policy-go.md) | Update-Policy für Go (Patch automatisch, Minor und Major von Hand) | accepted | 2026-10-31 |

ADR-0007 wurde am 2026-11-01 nach dem Review von Phase 3 in der Begründung korrigiert. Die Entscheidung selbst ist unverändert, das ADR bleibt als Dokument stehen.

## Vorlage

```markdown
# ADR-000N: <Titel>

Status: proposed | accepted | superseded by ADR-000M
Datum: <yyyy-mm-dd>

## Kontext
## Entscheidung
## Konsequenzen
```

Die Zeile `Überarbeitet: <yyyy-mm-dd>, <was geändert wurde>` ist optional. Sie steht im Kopf, wenn eine bestehende Entscheidung nach einem Review oder einem Plan neu begründet oder ergänzt wurde. Der Status bleibt dabei `accepted`.

## Status

- `proposed`: zur Diskussion, noch nicht umgesetzt.
- `accepted`: entschieden und im Code umgesetzt.
- `superseded by ADR-000M`: ersetzt durch eine neuere Entscheidung.

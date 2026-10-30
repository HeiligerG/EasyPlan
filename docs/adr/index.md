# Architekturentscheidungen

Ein ADR hält eine Entscheidung fest, die das Projekt prägt: Kontext, Entscheidung und Folgen. AdRs werden nummeriert und nie überschrieben. Eine spätere Entscheidung, die eine frühere ersetzt, bekommt eine eigene Nummer und verweist in der alten auf den Nachfolger.

| ADR | Titel | Status | Datum |
|---|---|---|---|
| [ADR-0001](0001-sqlite-ohne-cgo.md) | Eine SQLite-Datei ohne CGO (`modernc.org/sqlite`) | accepted | 2026-10-30 |
| [ADR-0002](0002-rappen-und-rundung.md) | Beträge als Rappen (`int64`), Rundung nur bei der Anzeige | accepted | 2026-10-30 |
| [ADR-0003](0003-kalendertage-utc.md) | Kalendertage als UTC-Mitternacht, Termine nur aus dem Startdatum | accepted | 2026-10-30 |
| [ADR-0004](0004-naechste-zahlung-pro-konto.md) | Nächste Zahlung pro Konto | accepted | 2026-10-30 |

## Vorlage

```markdown
# ADR-000N: <Titel>

Status: proposed | accepted | superseded by ADR-000M
Datum: <yyyy-mm-dd>

## Kontext
## Entscheidung
## Konsequenzen
```

## Status

- `proposed`: zur Diskussion, noch nicht umgesetzt.
- `accepted`: entschieden und im Code umgesetzt.
- `superseded by ADR-000M`: ersetzt durch eine neuere Entscheidung.

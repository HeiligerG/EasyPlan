# Berechnung

EasyPlan rechnet Fixkosten. Ausgaben laufen über Konten, jeder Eintrag gehört zu einem Konto und wiederholt sich nach einem bestimmten Rhythmus.

## Grundlage: Jahreskosten

Alles leitet sich aus den Kosten pro Jahr ab. Nur so passen Monat, Woche und Tag zueinander. Ein Jahr hat 365,25 Tage.

```text
Jahreskosten(Eintrag) =
    week:  Betrag * 365.25 / 7 / Intervall
    month: Betrag * 12 / Intervall
    year:  Betrag / Intervall
    once:  nicht enthalten
```

Daraus folgen die Perioden:

```text
Monat = Jahreskosten / 12
Woche = Jahreskosten * 7 / 365.25
Tag   = Jahreskosten / 365.25
```

Die Berechnung summiert zuerst die Jahreskosten und leitet Monat, Woche und Tag aus dieser Summe ab. Ein Konto mit 2'000 CHF Miete, 450 CHF Krankenkasse und 60 CHF Internet pro Monat ergibt 24'000 plus 5'400 plus 720 CHF pro Jahr, also 2'510 CHF pro Monat.

Die Funktion liefert ungerundete Werte in Rappen:

```go
need := plan.Totals(entries)       // ungerundet, in Rappen
month := plan.RoundCHF(need.Month) // 2510
```

## Rundung nur bei der Anzeige

Gerundet wird erst beim Anzeigen, auf ganze Franken, kaufmännisch (halbe Franken weg von null). `plan.RoundCHF` macht das, das Plan-Paket selbst rundet nie.

Weil die Anzeige jede Zeile für sich rundet, kann die Summe der angezeigten Kontowerte vom angezeigten Total abweichen. Im Beispiel unten ist das Total der Tageswerte 87 CHF, die Summe der drei Konto-Zeilen ist 86 CHF. Diese Abweichung von 1 CHF ist gewollt und wird nicht nachkorrigiert.

## Grenzen der Beträge

Ein Betrag ist immer positiv und höchstens 10^12 Rappen, also 10 Milliarden Franken. Die Grenze steht als `plan.MaxAmountMinor` im Code, damit die Summen der Beträge in den Berechnungen im Bereich von `int64` bleiben. Der Store prüft sie vor dem Schreiben, ein grösserer Betrag wird nicht gespeichert.

Die Grenzen für Intervall und Datum stehen unter [Wiederholungen](wiederholungen.md).

## Einmalige Ausgaben

Ein einmaliger Eintrag (`once`) ist kein laufender Bedarf. Er fliesst nicht in Monat, Woche und Tag ein, erscheint aber in der Timeline und in der Reserve. Ein Zahnarztbesuch am 12.11. für 300 CHF kostet damit im Monat November 0 CHF laufend, belastet aber die Reserve um 300 CHF, wenn der Termin im Zeitraum liegt.

## Aktive Einträge

Ein Eintrag gilt als aktiv, solange sein Enddatum `ends_on` nicht vor dem Stichtag liegt. `plan.ActiveOn` filtert entsprechend. Einmalige Einträge bleiben dabei erhalten, ihr Enddatum ignorieren die Berechnungen ohnehin.

## Reserve

```text
Reserve(Konto, N) = Summe aller Fälligkeiten des Kontos im Zeitraum [heute, heute + N Tage)
```

- Jede tatsächliche Fälligkeit zählt, auch wenn derselbe Eintrag im Zeitraum mehrfach vorkommt.
- Der Bereich ist halb offen: der heutige Tag zählt mit, der Tag `heute + N` nicht mehr.
- Die Reserve nutzt dieselbe Terminberechnung wie die Timeline, beide rufen `plan.Occurrences` auf. Termine entstehen nur in `internal/plan`, dort teilen sich `Occurrences` und `Next` den Kern `nth`.
- N ist 7, 30 (Standard), 60 oder 90. Die Auswahl steht auf dem [Dashboard](dashboard.md) und wird per Query-Parameter `n` gewählt, sie wird nicht gespeichert.

```go
reserve := plan.Reserve(entries, today, 30) // Rappen
```

## Nächste Zahlung

Die nächste Zahlung wird **pro Konto** ermittelt: der früheste Termin aller aktiven Einträge des Kontos, ab dem Stichtag, der Stichtag selbst eingeschlossen.

Der Eintrag mit dem frühesten Termin gewinnt, nicht der grösste Eintrag. Eine fällige Zahlung am heutigen Tag ist also die nächste Zahlung.

## Beispiel: Stand 1.10.2026

Die Ausgangsdaten:

| Konto | Eintrag | Betrag | Rhythmus |
|---|---|---:|---|
| Lifestyle | Miete | 2'000 | monatlich, am 1. |
| Lifestyle | Krankenkasse | 450 | monatlich, am 5. |
| Lifestyle | Internet | 60 | monatlich, am 15. |
| Sparen | Autoversicherung | 1'200 | jährlich, am 15.2. |
| Steuern | Serafe | 335 | jährlich, am 20.3. |

Die Ergebnisse am Stichtag 1.10.2026:

| Konto | Monat | Woche | Tag | Reserve 30 Tage | Nächste Zahlung |
|---|---:|---:|---:|---:|---|
| Lifestyle | 2'510 | 577 | 82 | 2'510 | Miete 01.10.2026 |
| Sparen | 100 | 23 | 3 | 0 | Autoversicherung 15.02.2027 |
| Steuern | 28 | 6 | 1 | 0 | Serafe 20.03.2027 |
| **Total** | **2'638** | **607** | **87** | **2'510** | |

Die Reserve über 30 Tage deckt den Bereich vom 1.10. bis zum 30.10.2026 ab. Für Lifestyle sind das die Miete am 1.10., die Krankenkasse am 5.10. und das Internet am 15.10., zusammen 2'510 CHF.

Das Total der Tageswerte ist 87, die Summe der Zeilen ist 86. Das ist die oben beschriebene Rundungsabweichung.

!!! note "Stand des Konzepts"
    `Konzept.md` wurde am 2026-10-30 an diese Regel angeglichen: Im Beispiel steht für Lifestyle jetzt „Miete 01.10.“, und die Regel der nächsten Zahlung ist dort beschrieben. `Konzept.md` bleibt das ursprüngliche Konzept, verbindlich für die Umsetzung sind der Code und diese Doku.

Die Werte sind mit dem Beispiel aus `Konzept.md` §5 im Test `internal/plan/example_concept_test.go` abgesichert.

## Weiterführend

- [Dashboard](dashboard.md): die Auswertung pro Konto und die Auswahl der Reserve
- [Wiederholungen](wiederholungen.md): Einheiten, Intervalle und Monatsende
- [ADR-0002: Rappen und Rundung](../adr/0002-rappen-und-rundung.md)
- [ADR-0004: Nächste Zahlung pro Konto](../adr/0004-naechste-zahlung-pro-konto.md)

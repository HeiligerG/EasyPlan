# Wiederholungen

Jeder Eintrag wiederholt sich nach einem Rhythmus. Der Rhythmus besteht aus einer Einheit und einem Intervall.

| Einheit | Bedeutung | Beispiel |
|---|---|---|
| `once` | einmalig am Startdatum | Zahnarzt am 12.11. |
| `week` | alle `interval` Wochen | `week` mit Intervall 2 = alle 2 Wochen |
| `month` | alle `interval` Monate | `month` mit Intervall 3 = quartalsweise |
| `year` | alle `interval` Jahre | Versicherung jährlich |

Das Intervall ist immer mindestens 1. Es steht in der Spalte `interval` der Tabelle `entries` und wird mit `CHECK (interval > 0)` abgesichert.

## Direktberechnung aus dem Startdatum

Der n-te Termin wird immer direkt aus dem Startdatum berechnet (`Startdatum + n * Intervall`), nie vom vorherigen Termin aus. Ein monatlicher Eintrag vom 31.01.2026 läuft deshalb so:

```text
31.01.2026 → 28.02.2026 → 31.03.2026 → 30.04.2026
```

Der 31.03.2026 entsteht aus dem Startdatum, nicht aus dem gekürzten 28.02.2026. Ein Eintrag bleibt so immer am selben Tag des Monats.

## Monatsende

Gibt es den Tag im Zielmonat nicht, wird der letzte Tag des Monats genommen. Im Schaltjahr gilt die Regel genauso:

```text
31.01.2028 → 29.02.2028 → 31.03.2028 → 30.04.2028
```

Dieselbe Regel greift bei jährlichen Einträgen, die am 29. Februar beginnen. Der Tag wird jedes Jahr neu aus dem Startdatum abgeleitet und auf das Monatsende gekürzt, bis wieder ein 29. Februar existiert:

```text
29.02.2028 → 28.02.2029 → 28.02.2030 → 28.02.2031 → 29.02.2032
```

## Enddatum

`ends_on` begrenzt den Eintrag. Der Termin am Enddatum zählt noch mit:

- Termin = `ends_on`: der Eintrag kommt im Zeitraum noch vor.
- Termin > `ends_on`: der Eintrag ist beendet und liefert keine Termine mehr.

Ein Enddatum vor dem Startdatum ist ungültig, `plan.Entry.Validate` lehnt solche Einträge ab. Ohne Enddatum läuft der Eintrag unbegrenzt weiter.

## Grenzen der Eingaben

`plan.Entry.Validate` prüft einen Eintrag, bevor er gespeichert wird. Der Store ruft die Prüfung bei jedem Schreiben auf, ungültige Einträge landen also nicht in der Datenbank.

| Grenze | Wert |
|---|---|
| Intervall | höchstens 10'000 Einheiten (`plan.MaxInterval`) |
| Startdatum und Enddatum | Jahre 1 bis 9999 |
| Enddatum | nicht vor dem Startdatum |

- Bei der Einheit `once` entfällt nur die obere Intervallgrenze, weil das Intervall dort nicht verwendet wird. Das Intervall muss weiterhin grösser als 0 sein.
- Die Datumswerte werden als Kalendertage verglichen, die Uhrzeit entscheidet nie über die Gültigkeit.
- Die Grenze für den Betrag steht unter [Berechnung](berechnung.md).

## Keine gespeicherten Termine

Einzelne Termine werden nicht in der Datenbank gespeichert. Die Datenbank enthält nur Startdatum, Einheit, Intervall und Enddatum. Alle Termine entstehen bei Bedarf aus diesen Angaben.

Ein einmaliger Eintrag, dessen Datum in der Vergangenheit liegt, gilt in der Liste als erledigt. Dafür wird nichts geschrieben. Die Prüfung übernimmt `plan.IsDone`:

```go
done := plan.IsDone(entry, today)
```

## Bereiche und Terminsuche

`plan.Occurrences` erzeugt alle Termine eines Eintrags im halb offenen Bereich `[from, to)`: der Starttag zählt mit, der Endtag nicht. `plan.Next` sucht den ersten Termin ab einem gegebenen Tag und benutzt dieselbe Formel, auch bei langen Intervallen wie alle drei Jahre.

`Occurrences` und `Next` teilen sich den Kern `nth`, in dem der n-te Termin aus dem Startdatum berechnet wird. `Next` greift dafür direkt auf `nth` zu, nicht auf `Occurrences`; `Reserve` und `Timeline` rufen `Occurrences` auf. Weil alle denselben Kern verwenden, liefern sie für denselben Eintrag und denselben Tag immer denselben Termin.

## Weiterführend

- [Berechnung](berechnung.md): Monat, Woche, Tag, Reserve und nächste Zahlung
- [ADR-0003: Kalendertage in UTC](../adr/0003-kalendertage-utc.md)

# Formate

Diese Seite beschreibt, wie EasyPlan Beträge und Daten schreibt, liest und anzeigt.

## Beträge in der Anzeige

Beträge werden in ganzen Franken angezeigt, kaufmännisch gerundet mit `plan.RoundCHF`: halbe Franken weg von null. Die Tausender trennt ein Apostroph.

```text
2'510
87
1'200'000
```

Die Anzeige rundet erst am Ende. Die Werte im Dashboard sind ungerundete Beträge in Rappen, die erst bei der Ausgabe auf ganze Franken kommen. Deshalb kann das Total vom 1 CHF von der Summe der Zeilen abweichen, siehe [Berechnung](berechnung.md) und [ADR-0002: Rappen und Rundung](../adr/0002-rappen-und-rundung.md).

!!! note "Der Apostroph im Quelltext"
    `html/template` escaped den Apostroph im Text. Im HTML-Quelltext steht darum `2&#39;510`, der Browser zeigt wieder `2'510`. Wer die Seite mit einem Werkzeug prüft, das den Quelltext zeigt, sieht die Entität und nicht das Zeichen.

## Beträge in der Eingabe

Das Formularfeld für den Betrag nimmt eine Zahl in Franken an. Der Parser liest sie als Text, damit kein Rundungsfehler einer Fliesstkommazahl entstehen kann.

### Trennzeichen

- `'` (Apostroph) und Leerzeichen sind Tausendertrenner und werden ignoriert.
- `.` (Punkt) und `,` (Komma) sind Dezimaltrenner.

Ist das **letzte** Trennzeichen ein Dezimaltrenner, darf ein früheres Trennzeichen der anderen Art ein Tausendertrenner sein. Damit sind beide Schreibweisen für Schweizer Zahlen gültig:

| Eingabe | Wert |
|---|---|
| `1,200.50` | 1200.50 |
| `1.200,50` | 1200.50 |
| `1'200.50` | 1200.50 |
| `1200` | 1200 |

Dasselbe Trennzeichen zweimal ist ungültig, weil dann nicht mehr klar ist, welches der Dezimaltrenner ist:

| Eingabe | Ergebnis |
|---|---|
| `1.2.3` | abgelehnt: „Der Betrag muss eine Zahl sein.“ |
| `1,2,3` | abgelehnt: „Der Betrag muss eine Zahl sein.“ |

### Nachkommastellen

Es sind höchstens zwei Nachkommastellen erlaubt. `1.200` hat drei und wird deshalb abgelehnt, auch wenn die Absicht ein Tausendertrenner war:

```text
1'200      ← richtig
1200       ← richtig
1.200      ← abgelehnt: „Höchstens 2 Nachkommastellen sind erlaubt.“
```

### Grenzen

- Der Betrag muss grösser als 0 sein. 0 und negative Beträge werden abgelehnt.
- Der Betrag darf höchstens `plan.MaxAmountMinor` betragen, also 10^12 Rappen oder 10 Milliarden Franken. Darüber heisst es „Der Betrag ist zu gross.“

## Daten

Ein Datum ist ein Kalendertag ohne Uhrzeit.

| Ort | Format | Beispiel |
|---|---|---|
| Anzeige | `TT.MM.JJJJ` | 01.10.2026 |
| Formularfeld | `JJJJ-MM-TT` | 2026-10-01 |
| Datenbank | `JJJJ-MM-TT` | 2026-10-01 |

Das Formularfeld ist ein `<input type="date">`. Der Browser zeigt es in der Form an, die der Browser kennt, und sendet den Wert im Format `JJJJ-MM-TT`. Die Datenbank kennt nur dieses Format, deshalb muss es nicht umgewandelt werden.

In der Anzeige der [Timeline](timeline.md) steht der Tag mit dem Wochentag und ohne Jahr, zum Beispiel „Do 01.10.“. Das Jahr steht in der Überschrift des Monats.

## Text in der Oberfläche

Namen und Notizen werden getrimmt, die Leerzeichen am Rand fallen weg. Ein Name darf höchstens 100 Zeichen haben, eine Notiz höchstens 1000.

Die Monatsnamen und die Wochentage der Timeline stehen als eigene Listen im Code, damit die Anzeige nicht von der Locale des Servers abhängt.

## Weiterführend

- [Berechnung](berechnung.md): Beträge in Rappen, Rundung und Grenzen
- [Wiederholungen](wiederholungen.md): Einheiten, Intervalle und das Enddatum

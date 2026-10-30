# ADR-0002: Beträge als Rappen (int64), Rundung nur bei der Anzeige

Status: accepted
Datum: 2026-10-30

## Kontext

EasyPlan rechnet Geldbeträge über viele Einträge. Beträge als Fliesstkommazahl zu speichern erzeugt Rundungsfehler, die sich über die Summen aufsummieren.

Gleichzeitig sollen Monat, Woche und Tag zueinander passen. Dazu leitet EasyPlan alles von den Jahreskosten ab und rundet erst bei der Anzeige. Daraus entsteht zwangsläufig eine Abweichung: Die Summe der angezeigten Kontowerte kann um 1 Franken vom angezeigten Total abweichen.

## Entscheidung

- Beträge werden als `int64` in Rappen gespeichert, in der Spalte `amount_minor`. Sie sind immer positiv, das Minuszeichen entsteht erst bei der Anzeige. Die Datenbank erzwingt das mit `CHECK (amount_minor > 0)`.
- Der Betrag eines Eintrags ist auf 10^12 Rappen begrenzt (`plan.MaxAmountMinor`), damit die Summen der Beträge in den Berechnungen im Bereich von `int64` bleiben.
- Die Jahreskosten rechnen in `float64`, weil sie den Bruch 365.25/7 brauchen. Das Ergebnis ist ein ungerundeter Wert in Rappen.
- `plan.Totals` summiert die ungerundeten Werte und liefert Monat, Woche und Tag ungerundet zurück.
- Gerundet wird erst bei der Anzeige, mit `plan.RoundCHF` auf ganze Franken, kaufmännisch (halbe Franken weg von null).
- Die Abweichung zwischen der Summe der Zeilen und dem Total wird nicht nachkorrigiert. Sie ist das erwartete Verhalten.

## Konsequenzen

- In der Datenbank entstehen keine Rundungsfehler, jeder Betrag ist exakt.
- Die Währung ist fest CHF. Ein Wechsel der Währung braucht eine neue Entscheidung, weil die Rappen-Zahl dann nichts mehr bedeutet.
- Wer die Anzeige nachrechnet, sieht eine Abweichung von bis zu 1 Franken. Das ist dokumentiert und gewollt, siehe [Berechnung](../features/berechnung.md).
- Die Anzeige formatiert die Beträge noch nicht, das folgt mit der Oberfläche in Phase 2.

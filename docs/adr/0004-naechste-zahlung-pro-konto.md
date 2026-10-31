# ADR-0004: Nächste Zahlung pro Konto

Status: accepted
Datum: 2026-10-30

## Kontext

Das Dashboard zeigt pro Konto die nächste Zahlung. Das Beispiel in `Konzept.md` §5 (Stand 1.10.2026) nannte für das Konto Lifestyle „Miete 01.11.“, obwohl im selben Beispiel die Krankenkasse am 5.10. und das Internet am 15.10. fällig sind und die Reserve am Stichtag die Miete vom 1.10. bereits mitzählt. Das Beispiel wurde am 2026-10-30 an die Entscheidung unten angeglichen.

Die Tabelle legt die Regel nicht fest. Drei Lesarten führen zu verschiedenen Ergebnissen:

- je Eintrag der nächste Termin ab morgen: Lifestyle zeigt „Miete 01.11.“, die Krankenkasse fehlt in der Übersicht des Kontos.
- frühester Termin aller Einträge des Kontos ab morgen: Lifestyle zeigt „Krankenkasse 05.10.“.
- frühester Termin nach dem Reservefenster: Lifestyle zeigt „Miete 01.11.“.

Ausserdem bleibt offen, ob ein heute fälliger Termin schon die nächste Zahlung ist.

## Entscheidung

Die nächste Zahlung ist **pro Konto** der früheste Termin aller aktiven Einträge des Kontos, ab dem Stichtag, der Stichtag selbst eingeschlossen. Formal: das Minimum aus `plan.Next(eintrag, heute)` über alle aktiven Einträge des Kontos.

- Ein heute fälliger Termin ist die nächste Zahlung.
- Der Betrag spielt keine Rolle. Der früheste Termin gewinnt, nicht der grösste Eintrag.
- Aktive Einträge sind die, deren Enddatum nicht vor dem Stichtag liegt.

Diese Regel passt zur Reserve und zur Timeline: beide schliessen den heutigen Tag ein und rufen `plan.Occurrences` auf. `plan.Next` liefert den Termin aus demselben Kern `nth`.

## Konsequenzen

- Am Stichtag 1.10.2026 zeigt das Konto Lifestyle „Miete 01.10.2026“, ab dem 2.10.2026 „Krankenkasse 05.10.2026“.
- Die Spalte „Nächste Zahlung“ im Beispiel von `Konzept.md` §5 wurde am 2026-10-30 an diese Entscheidung angeglichen, dort steht für Lifestyle jetzt „Miete 01.10.“. `Konzept.md` bleibt das ursprüngliche Konzept, verbindlich für die Umsetzung sind der Code und die Doku unter `docs/`. Die Regel steht auch unter [Berechnung](../features/berechnung.md).
- Die Regel wird mit dem Dashboard umgesetzt und dort getestet. `plan.Next` und `plan.ActiveOn` liegen dafür bereits vor. Die Anzeige steht unter [Dashboard](../features/dashboard.md).

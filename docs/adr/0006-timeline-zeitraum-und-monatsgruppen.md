# ADR-0006: Zeitraum und Monatsgruppen der Timeline

Status: accepted
Datum: 2026-10-31

## Kontext

`Konzept.md` §6 verlangt für die Timeline „die nächsten 12 Monate, nach Monat gruppiert, optional nach Konto filterbar“. Offen bleibt, wie die zwölf Monate und die Gruppen zusammenpassen.

Der Zeitraum der Reserve ist `[heute, heute + N Tage)`, und die Reserve schliesst den heutigen Tag ein. Ein Zeitraum `[heute, heute + 12 Monate)` ist damit die natürliche Wahl: er folgt derselben Logik, und ein heute fälliger Termin erscheint in der Timeline.

Die Gruppen sind eine Darstellungsfrage, keine Berechnungsfrage. Der Zeitraum `[heute, heute + 12 Monate)` enthält zwölf volle Monate plus den Rest des Monats, in dem er beginnt:

- Steht der heutige Tag auf dem 1. eines Monats, endet der Zeitraum am letzten Tag des zwölften Monats, und es sind genau zwölf Gruppen.
- An jedem anderen Tag endet der Zeitraum mitten in einem Monat, und es sind dreizehn Gruppen.

Ein erster Entwurf des Codes bildete immer genau zwölf feste Gruppen und liess Termine des dreizehnten Monats weg. Das war still: die Zahlungen verschwanden aus der Anzeige, ohne dass etwas gemeldet wurde. Der Fehler fiel erst auf, weil die Tests ein Stichtag auf dem 1. eines Monats benutzten.

## Entscheidung

- Der Zeitraum der Timeline ist der halb offene Bereich `[heute, heute + 12 Monate)`.
- Die Gruppen laufen vom Monat des heutigen Tages bis zum Monat des letzten Tages des Zeitraums. Das sind am 1. eines Monats zwölf Gruppen, an jedem anderen Tag dreizehn.
- Der letzte Monat zeigt nur den Teil, der im Zeitraum liegt.
- Ein Monat ohne Zahlung wird mit dem Hinweis „Keine Zahlungen in diesem Monat.“ gezeigt. Die Gruppen decken den ganzen Zeitraum ab, es gibt keine Lücke und keine Zahlung ausserhalb der Gruppen.
- Die Termine entstehen in `internal/plan` über `plan.Timeline`. Die Web-Schicht berechnet keinen Termin selbst und verwirft auch keinen. Ein Termin, der ausserhalb der Gruppen läge, wäre ein Fehler und wird als Fehler gemeldet, nicht stillschweigend weggelassen.

## Konsequenzen

- Die Timeline zeigt am 1. eines Monats zwölf Gruppen, an jedem anderen Tag dreizehn. Der Umfang schwankt dadurch mit dem Stichtag, aber die Abdeckung des Zeitraums ist immer vollständig.
- Die Termine werden nach Tag sortiert. Innerhalb eines Tages entscheidet die Reihenfolge der Konten, dann der Name des Eintrags und dann seine Kennung. Die Reihenfolge ist damit stabil und unabhängig von der Reihenfolge der Datenbank.
- Die Tests decken Stichtage am 1., an einem anderen Tag und am 29.02. eines Schaltjahres ab, damit die Gruppenzahl und die Abdeckung des Zeitraums nicht auseinanderlaufen.
- Die Reserve rechnet mit demselben Bereich und derselben Terminberechnung. Beide nutzen `plan.Occurrences`, damit Dashboard und Timeline nicht auseinanderlaufen.

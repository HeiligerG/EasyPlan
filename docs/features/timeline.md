# Timeline

Die Timeline zeigt, welche Zahlungen in den nächsten zwölf Monaten anfallen, nach Monat gruppiert. Die Adresse ist `/timeline`.

## Der Zeitraum

Der Zeitraum ist der halb offene Bereich `[heute, heute + 12 Monate)`:

- Der heutige Tag zählt mit.
- Der Tag genau zwölf Monate später zählt nicht mehr.

Die Termine entstehen in `internal/plan`, die Web-Schicht rechnet keinen einzigen Termin selbst. Sie ruft `plan.Timeline` auf und gruppiert das Ergebnis nur noch nach Monat.

## Die Monatsgruppen

Die Gruppen laufen vom Monat des heutigen Tages bis zum Monat des letzten Tages des Zeitraums. Daraus folgen zwei Fälle:

- Steht der heutige Tag auf dem 1. eines Monats, sind es genau **12 Gruppen**.
- An jedem anderen Tag sind es **13 Gruppen**. Der letzte Monat zeigt dann nur noch den Rest des Monats, in dem der Zeitraum endet.

Am 15.10.2026 reicht der Zeitraum bis zum 14.10.2027. Die Gruppen laufen von Oktober 2026 bis Oktober 2027, und die Gruppe Oktober 2027 enthält nur die Zahlungen bis zum 14.10.2027.

Ein Monat ohne Zahlung wird trotzdem gezeigt, mit dem Hinweis „Keine Zahlungen in diesem Monat.“ So bleibt sichtbar, dass der Monat geprüft wurde.

Gibt es im ganzen Zeitraum keine einzige Zahlung, zeigt die Seite statt der Gruppen einen Hinweis mit dem Link auf die [Einträge](eintraege.md).

## Die Summe des Monats

Unter den Zahlungen eines Monats steht die Summe der angezeigten Ereignisse. Die Summe bezieht sich nur auf die Zahlungen, die in dieser Ansicht stehen: mit einem Kontofilter ist es die Summe dieses Kontos, ohne Filter die Summe aller Konten.

## Der Kontofilter

Über der Liste steht eine Auswahl mit allen aktiven Konten und dem Eintrag „Alle Konten“. Die Auswahl ist ein normales Formular mit der Methode `get`, sie braucht kein JavaScript. Der gewählte Konto steht als Query-Parameter in der Adresse:

```text
/timeline?account=2
```

- Ohne den Parameter zeigt die Seite alle Konten.
- Ein Wert, der keine Zahl ist, ein Wert 0 oder ein negativer Wert bedeutet „alle Konten“.
- Eine unbekannte Kennung oder ein deaktiviertes Konto bedeutet ebenfalls „alle Konten“. Die Seite antwortet nie mit einem Fehler.

## Die Reihenfolge

Die Zahlungen stehen nach Tag. An einem Tag entscheidet die Reihenfolge der Konten, also die Reihenfolge, die auf der Seite [Konten](konten.md) eingestellt ist. Bei gleichem Konto entscheidet der Name des Eintrags, bei gleichem Namen die Kennung des Eintrags. Die Reihenfolge ist damit immer dieselbe.

## Die Anzeige

Jede Zahlung zeigt den Tag mit dem Wochentag, den Betrag, den Namen des Eintrags und den Namen des Kontos:

```text
Do 01.10.    2'000
Miete
Lifestyle
```

Das Jahr steht in der Überschrift des Monats, darum fehlt es beim Tag. Der Tag steht zusätzlich als `datetime`-Attribut im Format `JJJJ-MM-TT` im Quelltext.

## Was in der Timeline steht

- Nur aktive Konten und aktive Einträge. Die Einträge eines deaktivierten Kontos und beendete Einträge erscheinen nicht.
- Ein einmaliger Eintrag erscheint an seinem Tag, wenn er in den Zeitraum fällt.
- Ein Eintrag, dessen Enddatum vor dem heutigen Tag liegt, kommt nicht mehr vor.

## Weiterführend

- [Berechnung](berechnung.md): die Reserve rechnet mit demselben Zeitraum und denselben Terminen
- [Konten](konten.md): die Reihenfolge, nach der die Zahlungen eines Tages sortiert werden

# Dashboard

Das Dashboard ist die Startseite der App. Es beantwortet die drei Fragen von EasyPlan auf einen Blick: Wie viel Geld brauche ich pro Monat, Woche und Tag? Wie viel muss pro Konto in den nächsten N Tagen vorhanden sein? Welche Zahlung kommt als Nächstes?

Die Adresse ist `/`.

## Die Tabelle

Für jedes aktive Konto steht eine Zeile, darunter eine Totalzeile:

| Spalte | Inhalt |
|---|---|
| Konto | Name des Kontos |
| Monat | Bedarf pro Monat, auf ganze Franken gerundet |
| Woche | Bedarf pro Woche, auf ganze Franken gerundet |
| Tag | Bedarf pro Tag, auf ganze Franken gerundet |
| Reserve N Tage | Summe aller Fälligkeiten des Kontos in den nächsten N Tagen |
| Nächste Zahlung | Name des Eintrags und Tag des frühesten noch kommenden Termins |

Die Werte stammen aus derselben Berechnung wie unter [Berechnung](berechnung.md). Die Tabelle wird bei jedem Aufruf neu berechnet, es werden keine Zwischenergebnisse gespeichert.

### Das Total weicht von der Zeilensumme ab

Das Total wird aus den ungerundeten Werten aller aktiven Einträge berechnet, die Zeilen werden einzeln gerundet. Deshalb kann das angezeigte Total um 1 CHF von der Summe der angezeigten Zeilen abweichen. Das ist gewollt und wird nicht nachkorrigiert, siehe [ADR-0002: Rappen und Rundung](../adr/0002-rappen-und-rundung.md).

Im Beispiel aus [Berechnung](berechnung.md) ist das Total der Tageswerte 87 CHF, die Summe der drei Konto-Zeilen ist 86 CHF.

## Die Auswahl der Reserve

Über der Tabelle stehen vier Links: 7, 30, 60 und 90 Tage. Der Link setzt den Query-Parameter `n`:

```text
/?n=7
```

- Erlaubt sind 7, 30, 60 und 90 Tage.
- 30 Tage ist der Standard und wird verwendet, wenn der Parameter fehlt.
- Ein unbekannter Wert fällt auf 30 Tage zurück. Die Seite antwortet also nie mit einem Fehler.
- Die Auswahl wird nicht gespeichert. Sie gilt für den einen Aufruf der Seite.

Die Reserve ist der halb offene Bereich `[heute, heute + N Tage)`: der heutige Tag zählt mit, der Tag `heute + N` nicht mehr.

## Die nächste Zahlung

Die nächste Zahlung ist **pro Konto** der früheste Termin aller aktiven Einträge des Kontos, ab dem heutigen Tag, der heutige Tag selbst eingeschlossen. Ein heute fälliger Termin ist also die nächste Zahlung.

Fällt der Betrag auf den gleichen Tag, gewinnt der Eintrag mit dem kleineren Namen, bei gleichen Namen der Eintrag mit der kleineren Kennung. Die Anzeige ist damit eindeutig.

Die Regel steht im Detail unter [ADR-0004: Nächste Zahlung pro Konto](../adr/0004-naechste-zahlung-pro-konto.md).

## Was auf dem Dashboard zählt

- Deaktivierte Konten erscheinen nicht als Zeile und ihre Einträge fliessen nicht in das Total ein.
- Ein wiederkehrender Eintrag, dessen Enddatum vor dem heutigen Tag liegt, zählt nicht mit.
- Ein einmaliger Eintrag zählt nicht in Monat, Woche und Tag, aber in der Reserve, wenn sein Termin im Zeitraum liegt. Liegt sein Datum in der Vergangenheit, zählt er nirgends mit.

## Leerzustand

Sind keine aktiven Einträge vorhanden, zeigt das Dashboard statt der Tabelle einen Hinweis mit dem Link auf die [Einträge](eintraege.md).

## Beispiel: Stand 1.10.2026

Die Ausgangsdaten:

| Konto | Eintrag | Betrag | Rhythmus |
|---|---|---:|---|
| Lifestyle | Miete | 2'000 | monatlich, am 1. |
| Lifestyle | Krankenkasse | 450 | monatlich, am 5. |
| Lifestyle | Internet | 60 | monatlich, am 15. |
| Sparen | Autoversicherung | 1'200 | jährlich, am 15.2. |
| Steuern | Serafe | 335 | jährlich, am 20.3. |

Das Dashboard am Stichtag 1.10.2026 mit der Auswahl 30 Tage:

| Konto | Monat | Woche | Tag | Reserve 30 Tage | Nächste Zahlung |
|---|---:|---:|---:|---:|---|
| Lifestyle | 2'510 | 577 | 82 | 2'510 | Miete 01.10.2026 |
| Sparen | 100 | 23 | 3 | 0 | Autoversicherung 15.02.2027 |
| Steuern | 28 | 6 | 1 | 0 | Serafe 20.03.2027 |
| **Total** | **2'638** | **607** | **87** | **2'510** | |

Die Reserve über 30 Tage deckt den Bereich vom 1.10. bis zum 30.10.2026 ab. Für Lifestyle sind das die Miete am 1.10., die Krankenkasse am 5.10. und das Internet am 15.10., zusammen 2'510 CHF.

## Weiterführend

- [Berechnung](berechnung.md): Monat, Woche, Tag, Reserve und Rundung
- [Einträge](eintraege.md): die Daten, aus denen das Dashboard rechnet
- [Formate](formate.md): wie Beträge und Daten angezeigt werden

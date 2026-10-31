# Konten

Ein Konto fasst die Einträge zusammen, die in dieselbe Kategorie gehören. Jeder Eintrag gehört zu genau einem Konto, und die Auswertungen des [Dashboards](dashboard.md) und der [Timeline](timeline.md) rechnen pro Konto.

Die Adresse ist `/accounts`.

## Die Liste

Die Liste zeigt alle Konten in ihrer Reihenfolge: zuerst die aktiven, dahinter die deaktivierten, jeweils in der Reihenfolge, die auf der Seite eingestellt ist. Zu jedem Konto stehen sein Status, die Anzahl seiner Einträge und die Aktionen.

## Die Aktionen

| Aktion | Wirkung |
|---|---|
| Neues Konto | Legt ein Konto mit dem eingegebenen Namen an. |
| Umbenennen | Öffnet ein Formular und speichert den neuen Namen. |
| Nach oben / Nach unten | Verschiebt das Konto um eine Position. Die Reihenfolge der Konten wird neu nummeriert, sie bleibt lückenlos. |
| Deaktivieren / Aktivieren | Nimmt das Konto aus den Auswertungen heraus oder nimmt es wieder hinein. |

Nach jeder Aktion antwortet der Server mit dem Status 303 und der Adresse der Liste.

Die Schaltflächen „Nach oben“ und „Nach unten“ sind beim ersten und beim letzten Konto deaktiviert, weil das Verschieben dort nichts ändern würde.

### Deaktivieren statt Löschen

In der Oberfläche gibt es keine Möglichkeit, ein Konto zu löschen. Deaktivieren ist umkehrbar, Löschen nicht, und die Einträge eines Kontos sollen nicht versehentlich verschwinden.

Der Datenspeicher kann ein Konto ohne Einträge trotzdem löschen (`store.DeleteAccount`). Die Oberfläche ruft diese Funktion nicht auf, das `Store`-Interface der Web-Schicht enthält sie nicht.

Ein Konto mit Einträgen lässt sich nur deaktivieren. Auch das Löschen eines leeren Kontos ist in der Oberfläche nicht vorgesehen.

### Die Wirkung der Deaktivierung

Ein deaktiviertes Konto zählt nicht im Dashboard, nicht in der Reserve und nicht in der Timeline. Seine Einträge bleiben in der [Einträge-Liste](eintraege.md) sichtbar und können weiterhin bearbeitet werden.

## Der Name

- Der Name wird getrimmt, Leerzeichen am Anfang und am Ende fallen weg.
- Der Name muss zwischen 1 und 100 Zeichen lang sein.
- Der Name muss eindeutig sein. Ein bereits vergebener Name wird mit „Dieser Name ist bereits vergeben.“ abgelehnt.
- Die Gross- und Kleinschreibung zählt: „Lifestyle“ und „lifestyle“ sind zwei verschiedene Namen.

Ein leerer Name wird mit „Der Name darf nicht leer sein.“ abgelehnt. Die Prüfung passiert serverseitig; das Formular antwortet bei einem Fehler mit dem Status 422 und zeigt den eingegebenen Namen noch einmal.

## Die Standardkonten

Eine neue Datenbank bekommt mit der ersten Migration drei Standardkonten:

| Name | Reihenfolge |
|---|---:|
| Lifestyle | 1 |
| Sparen | 2 |
| Steuern | 3 |

Sie lassen sich umbenennen, umsortieren und deaktivieren. Ein gelöschtes Standardkonto kommt nicht zurück. Die Einzelheiten stehen unter [Datenspeicher](datenspeicher.md).

## Weiterführend

- [Einträge](eintraege.md): die Einträge eines Kontos
- [Datenspeicher](datenspeicher.md): Schema, Standardkonten und Fehler des Stores

---
description: Proaktive Ableitung globaler Skills, MCPs, Workflows und Regeln aus Projekterkenntnissen. Wird geladen, wenn der Agent eine Verbesserungsmöglichkeit erkennt.
---

# Proaktive Ableitung globaler Patterns

## Zweck

EasyPlan dient nicht nur als Endprodukt, sondern auch als **Quelle für wiederverwendbare globale Konfigurationen** (Skills, MCPs, Workflows, Regeln), die aus den hier gemachten Erfahrungen entstehen. Andere Projekte sollen von diesen Erkenntnissen profitieren, ohne dass sie neu erarbeitet werden müssen.

## Auslöser

Der Workflow wird geladen, sobald mindestens einer der folgenden Punkte zutrifft:

- Eine **nicht‑triviale Entscheidung** getroffen wird (z. B. neuer Branch‑Workflow, neue Validierung, neues Tooling).
- Ein **wiederkehrendes Problem** gelöst wurde, das auch in anderen Projekten auftreten könnte.
- Eine **Konvention etabliert** wird (z. B. Conventional Commits, Sprint‑Branches, Geld in Minor Units).
- Eine **Automatisierung** möglich wird, die bisher manuell erfolgte.
- Ein **externer Dienst** (z. B. Gotify, Discord, Telegram) sinnvoll eingebunden wird und das Integrations­muster verallgemeinerbar ist.

## Vorgehen

1. **Erkennen** – Der Agent erkennt ein potenziell generalisierbares Pattern.
2. **Vorschlagen** – Er schlägt dem Benutzer vor, daraus einen globalen Skill, MCP, Workflow oder Regel abzuleiten, und nennt den konkreten Zielpfad.
3. **Konsens** – Der Benutzer entscheidet, ob das Pattern extrahiert wird.
4. **Anlegen** am passenden Ort:
   - Globale Skills → `~/.config/kilo/skills/<name>/SKILL.md`
   - Globale MCPs → `~/.config/kilo/mcp.json` (oder projektspezifisch `.kilo/mcp.json`)
   - Globale Regeln → `~/.config/kilo/rules/` (oder projektspezifisch `.kilo/rules/`)
   - Lokale Workflows → `.kilo/workflow/<name>.md` (im jeweiligen Projekt)
5. **Dokumentieren** – Im Commit‑Body oder PR‑Body vermerken, dass das Pattern aus EasyPlan stammt und auf welches globale Ziel es zeigt.
6. **Iterieren** – Bei späteren Sessions auf das Pattern zurückgreifen und ggf. verfeinern oder in andere Projekte übertragen.

## Beispiele aus EasyPlan

| Pattern | Abgeleitet als |
|---------|----------------|
| Hierarchischer Git‑Workflow (Sprint‑Branch → Sub‑Branch → PR) | Globaler Skill `git-workflow` |
| Bedarf an projekt­spezifischem `CODE_OF_CONDUCT` | Globaler Skill `code-of-conduct-generator` |
| Go‑Lint‑Kette (`gofmt` + `go vet` + `golangci-lint`) | Lokaler Workflow `lint-go.md` (geplant) |
| Notifikations‑Dispatcher mit mehreren Webhook‑Zielen | Globaler Skill `notification-dispatcher` (geplant) |

## Hinweise

- Nur **nicht‑triviale, wiederverwendbare** Patterns extrahieren – projektspezifische Details (z. B. Datenbank‑Schema, Geschäftslogik) bleiben im Projekt.
- Vorschläge immer **mit Begründung** und **konkretem Pfad** machen.
- Vor dem Anlegen eines globalen Elements **explizit bestätigen lassen** – globale Skills wirken in *allen* Projekten.
- Bei Sicherheits‑/Datenschutz‑Relevanz (z. B. Token‑Handhabung) immer das globale Pattern so gestalten, dass es **sichere Defaults** erzwingt.
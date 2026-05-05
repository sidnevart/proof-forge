import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { FaqAccordion } from "../faq-accordion";

const ITEMS = [
  { q: "ВОПРОС ОДИН", a: "Ответ на первый вопрос." },
  { q: "ВОПРОС ДВА", a: "Ответ на второй вопрос." },
  { q: "ВОПРОС ТРИ", a: "Ответ на третий вопрос." },
  { q: "ВОПРОС ЧЕТЫРЕ", a: "Ответ на четвёртый вопрос." },
];

describe("FaqAccordion", () => {
  it("renders all questions", () => {
    render(<FaqAccordion items={ITEMS} />);
    for (const item of ITEMS) {
      expect(screen.getByText(item.q)).toBeInTheDocument();
    }
  });

  it("first item is expanded by default", () => {
    render(<FaqAccordion items={ITEMS} />);
    const firstBtn = screen.getByRole("button", { name: /ВОПРОС ОДИН/ });
    expect(firstBtn).toHaveAttribute("aria-expanded", "true");
  });

  it("other items are collapsed by default", () => {
    render(<FaqAccordion items={ITEMS} />);
    const secondBtn = screen.getByRole("button", { name: /ВОПРОС ДВА/ });
    const thirdBtn = screen.getByRole("button", { name: /ВОПРОС ТРИ/ });
    expect(secondBtn).toHaveAttribute("aria-expanded", "false");
    expect(thirdBtn).toHaveAttribute("aria-expanded", "false");
  });

  it("clicking a collapsed item opens it and closes the previously open one", async () => {
    const user = userEvent.setup();
    render(<FaqAccordion items={ITEMS} />);

    const firstBtn = screen.getByRole("button", { name: /ВОПРОС ОДИН/ });
    const secondBtn = screen.getByRole("button", { name: /ВОПРОС ДВА/ });

    expect(firstBtn).toHaveAttribute("aria-expanded", "true");
    expect(secondBtn).toHaveAttribute("aria-expanded", "false");

    await user.click(secondBtn);

    expect(firstBtn).toHaveAttribute("aria-expanded", "false");
    expect(secondBtn).toHaveAttribute("aria-expanded", "true");
  });

  it("keyboard Enter toggles an item", async () => {
    const user = userEvent.setup();
    render(<FaqAccordion items={ITEMS} />);

    const secondBtn = screen.getByRole("button", { name: /ВОПРОС ДВА/ });
    secondBtn.focus();

    await user.keyboard("{Enter}");
    expect(secondBtn).toHaveAttribute("aria-expanded", "true");

    await user.keyboard("{Enter}");
    expect(secondBtn).toHaveAttribute("aria-expanded", "false");
  });

  it("keyboard Space toggles an item", async () => {
    const user = userEvent.setup();
    render(<FaqAccordion items={ITEMS} />);

    const thirdBtn = screen.getByRole("button", { name: /ВОПРОС ТРИ/ });
    thirdBtn.focus();

    await user.keyboard(" ");
    expect(thirdBtn).toHaveAttribute("aria-expanded", "true");
  });
});

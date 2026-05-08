import { describe, expect, it } from "vitest";

import { metadata } from "./layout";

describe("app layout metadata", () => {
  it("uses russian product title and description", () => {
    expect(metadata.title).toBe("ProofForge — среда где движение становится нормой");
    expect(metadata.description).toBe(
      "Сильная среда. Здоровая конкуренция. Не планируешь — делаешь. Проверено кругом людей рядом.",
    );
  });
});

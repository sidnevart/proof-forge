import { describe, expect, it } from "vitest";

import { metadata } from "./layout";

describe("app layout metadata", () => {
  it("uses russian product title and description", () => {
    expect(metadata.title).toBe("ProofForge — круг друзей, где не сдать = заморозка");
    expect(metadata.description).toBe(
      "Соревнуйся с друзьями за дисциплину. Сдавай пруф каждый день. Пропустил — все видят.",
    );
  });
});

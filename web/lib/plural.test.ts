import { describe, expect, it } from "vitest";

import { pluralizeRu } from "./plural";

const forms: [string, string, string] = ["участник", "участника", "участников"];

describe("pluralizeRu", () => {
  it("uses 'many' for 0", () => {
    expect(pluralizeRu(0, forms)).toBe("участников");
  });

  it("uses 'one' for 1", () => {
    expect(pluralizeRu(1, forms)).toBe("участник");
  });

  it("uses 'few' for 2-4", () => {
    expect(pluralizeRu(2, forms)).toBe("участника");
    expect(pluralizeRu(3, forms)).toBe("участника");
    expect(pluralizeRu(4, forms)).toBe("участника");
  });

  it("uses 'many' for 5-20", () => {
    expect(pluralizeRu(5, forms)).toBe("участников");
    expect(pluralizeRu(11, forms)).toBe("участников");
    expect(pluralizeRu(12, forms)).toBe("участников");
    expect(pluralizeRu(14, forms)).toBe("участников");
    expect(pluralizeRu(20, forms)).toBe("участников");
  });

  it("uses 'one' for 21, 'few' for 22-24, 'many' for 25", () => {
    expect(pluralizeRu(21, forms)).toBe("участник");
    expect(pluralizeRu(22, forms)).toBe("участника");
    expect(pluralizeRu(24, forms)).toBe("участника");
    expect(pluralizeRu(25, forms)).toBe("участников");
  });

  it("uses 'one' for 101", () => {
    expect(pluralizeRu(101, forms)).toBe("участник");
  });

  it("treats negative numbers by absolute value", () => {
    expect(pluralizeRu(-1, forms)).toBe("участник");
    expect(pluralizeRu(-5, forms)).toBe("участников");
  });

  it("truncates fractional numbers", () => {
    expect(pluralizeRu(1.7, forms)).toBe("участник");
    expect(pluralizeRu(2.9, forms)).toBe("участника");
  });
});

import { LandingNav } from "@/components/marketing/landing-nav";
import { LandingPage } from "@/components/marketing/landing-page";

export default function HomePage() {
  // Marketing surfaces are full-bleed on desktop: backgrounds, borders, and
  // section dividers stretch edge-to-edge while text stays inside an inner
  // container managed by each section's own CSS.
  return (
    <>
      <LandingNav />
      <LandingPage />
    </>
  );
}

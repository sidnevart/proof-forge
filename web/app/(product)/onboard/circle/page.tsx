import { redirect } from "next/navigation";

// Backward compat: /onboard/circle now redirects to /onboard/goal
export default function OnboardCirclePage() {
  redirect("/onboard/goal");
}

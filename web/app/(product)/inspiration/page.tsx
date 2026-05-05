import { redirect } from "next/navigation";

export default function InspirationPage() {
  redirect("/feed?tab=similar");
}

"use client";

import Link from "next/link";

import type { TeamDetail, TeamRole } from "@/lib/types";
import styles from "./teams-list.module.css";

interface TeamsListProps {
  teams: TeamDetail[];
  /** Empty-state CTA href. Default: `/teams/new`. */
  emptyCtaHref?: string;
  /** Empty-state CTA label. Default: «СОЗДАТЬ КОМАНДУ». */
  emptyCtaLabel?: string;
}

/**
 * Renders a list of team cards, one per active membership. Each card links to
 * `/teams/:id`. Shows a my_role badge ("ТИМЛИД" / "ДОВЕРЕННЫЙ" / "УЧАСТНИК")
 * so the user can tell at a glance which teams they own.
 *
 * Empty state shows a CTA button. Default points at /teams/new but the parent
 * can override (e.g. point at the join page when the empty list comes from a
 * search).
 */
export function TeamsList({
  teams,
  emptyCtaHref = "/teams/new",
  emptyCtaLabel = "СОЗДАТЬ КОМАНДУ",
}: TeamsListProps) {
  if (teams.length === 0) {
    return (
      <div className={styles.empty}>
        <span className={styles.emptyText}>КОМАНД ПОКА НЕТ.</span>
        <Link href={emptyCtaHref} className={styles.emptyCta}>
          {emptyCtaLabel} →
        </Link>
      </div>
    );
  }

  return (
    <ul className={styles.list}>
      {teams.map((d) => (
        <li key={d.team.id} className={styles.item}>
          <Link href={`/teams/${d.team.id}`} className={styles.card}>
            <span className={styles.name}>{d.team.name.toUpperCase()}</span>
            <span className={styles.meta}>
              <span className={styles.role}>{roleLabel(d.my_membership.role)}</span>
              <span className={styles.dot}>·</span>
              <span className={styles.count}>{d.member_count} чел.</span>
              {d.team.archived_at && <span className={styles.archived}>· В АРХИВЕ</span>}
            </span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

function roleLabel(role: TeamRole): string {
  switch (role) {
    case "lead":
      return "ТИМЛИД";
    case "trusted_approver":
      return "ДОВЕРЕННЫЙ";
    case "member":
      return "УЧАСТНИК";
  }
}

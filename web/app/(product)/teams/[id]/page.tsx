"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState, useCallback } from "react";

import {
  ApiError,
  addProofComment,
  approveProof,
  archiveTeam,
  assembleProofWithAI,
  freezeDailyLog,
  getDailyLogStreak,
  getLeadBriefing,
  getProofComments,
  getTeam,
  getTeamFeed,
  leaveTeam,
  listDailyLogs,
  regenerateTeamInvite,
  rejectProof,
  setMyAIConsent,
  submitDailyLog,
  trackEvent,
  type FeedItem,
} from "@/lib/api";
import type { AssembleProofResult, DailyLogEntry, DailyLogStreak, LeadBriefing, ProofComment, TeamDetail } from "@/lib/types";

import styles from "./page.module.css";

type State =
  | { kind: "loading" }
  | { kind: "ready"; detail: TeamDetail }
  | { kind: "error"; message: string };

export default function TeamDetailPage() {
  const params = useParams<{ id: string }>();
  const teamId = Number(params?.id);
  const router = useRouter();

  const [state, setState] = useState<State>({ kind: "loading" });
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  const [streak, setStreak] = useState<DailyLogStreak | null>(null);
  const [entries, setEntries] = useState<DailyLogEntry[]>([]);
  const [logText, setLogText] = useState("");
  const [logUrl, setLogUrl] = useState("");
  const [logBusy, setLogBusy] = useState(false);

  const [feedItems, setFeedItems] = useState<FeedItem[]>([]);
  const [feedCursor, setFeedCursor] = useState<number | null>(null);
  const [feedBusy, setFeedBusy] = useState(false);
  const [expandedProofId, setExpandedProofId] = useState<number | null>(null);
  const [commentsMap, setCommentsMap] = useState<Record<number, ProofComment[]>>({});
  const [commentTextMap, setCommentTextMap] = useState<Record<number, string>>({});
  const [commentBusyMap, setCommentBusyMap] = useState<Record<number, boolean>>({});
  const [rejectProofId, setRejectProofId] = useState<number | null>(null);
  const [rejectText, setRejectText] = useState("");
  const [rejectBusy, setRejectBusy] = useState(false);

  const [aiConsent, setAiConsent] = useState<boolean | null>(null);
  const [assembleResult, setAssembleResult] = useState<AssembleProofResult | null>(null);
  const [assembleBusy, setAssembleBusy] = useState(false);
  const [briefing, setBriefing] = useState<LeadBriefing | null>(null);
  const [briefingTarget, setBriefingTarget] = useState<number | null>(null);

  useEffect(() => {
    if (!Number.isFinite(teamId) || teamId <= 0) {
      setState({ kind: "error", message: "Команда не найдена." });
      return;
    }
    getTeam(teamId)
      .then((detail) => setState({ kind: "ready", detail }))
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.code === "team.not_member") {
          setState({ kind: "error", message: "Ты не в этой команде." });
        } else {
          setState({
            kind: "error",
            message: err instanceof ApiError ? err.message : "Не удалось загрузить команду.",
          });
        }
      });
  }, [teamId]);

  useEffect(() => {
    if (!Number.isFinite(teamId) || teamId <= 0) return;
    const today = new Date().toISOString().slice(0, 10);
    const weekAgo = new Date(Date.now() - 6 * 86400000).toISOString().slice(0, 10);
    getDailyLogStreak(teamId).then(setStreak).catch(() => setStreak(null));
    listDailyLogs(teamId, weekAgo, today).then((d) => setEntries(d.entries)).catch(() => setEntries([]));
  }, [teamId]);

  const loadFeed = useCallback(async (cursor: number) => {
    if (feedBusy) return;
    setFeedBusy(true);
    try {
      const res = await getTeamFeed(teamId, cursor > 0 ? cursor : undefined);
      if (cursor === 0) {
        setFeedItems(res.items);
      } else {
        setFeedItems((prev) => [...prev, ...res.items]);
      }
      setFeedCursor(res.next_cursor ?? null);
    } catch {
      // silently fail — feed is not critical
    } finally {
      setFeedBusy(false);
    }
  }, [teamId, feedBusy]);

  useEffect(() => {
    if (!Number.isFinite(teamId) || teamId <= 0) return;
    loadFeed(0);
    trackEvent("team_feed_opened", { source_surface: "web", team_id: teamId }, teamId).catch(() => {
      // silently fail
    });
  }, [teamId, loadFeed]);

  useEffect(() => {
    if (state.kind === "ready") {
      setAiConsent(state.detail.my_membership.ai_consent);
    }
  }, [state]);

  if (state.kind === "loading") {
    return (
      <div className={`page-shell ${styles.page}`}>
        <div className={styles.loading}>ЗАГРУЖАЕМ…</div>
      </div>
    );
  }
  if (state.kind === "error") {
    return (
      <div className={`page-shell ${styles.page}`}>
        <div className={styles.error}>
          {state.message}{" "}
          <Link href="/teams" className={styles.errorLink}>
            ← КОМАНДЫ
          </Link>
        </div>
      </div>
    );
  }

  const { detail } = state;
  const team = detail.team;
  const isLead = detail.my_membership.role === "lead";

  async function handleCopy() {
    if (!team.invite_code) return;
    try {
      await navigator.clipboard.writeText(team.invite_code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // fallback: do nothing — user can select+copy manually
    }
  }

  async function handleRegenerate() {
    if (busy) return;
    if (!window.confirm("Перегенерировать код приглашения? Старый перестанет работать.")) {
      return;
    }
    setBusy(true);
    try {
      const code = await regenerateTeamInvite(teamId);
      setState({
        kind: "ready",
        detail: { ...detail, team: { ...team, invite_code: code } },
      });
    } catch (err) {
      window.alert(err instanceof ApiError ? err.message : "Не удалось перегенерировать код.");
    } finally {
      setBusy(false);
    }
  }

  async function handleArchive() {
    if (busy) return;
    if (!window.confirm("Архивировать команду? Новые участники не смогут вступить.")) {
      return;
    }
    setBusy(true);
    try {
      await archiveTeam(teamId);
      router.push("/teams");
    } catch (err) {
      window.alert(err instanceof ApiError ? err.message : "Не удалось архивировать.");
      setBusy(false);
    }
  }

  async function handleLeave() {
    if (busy) return;
    if (!window.confirm("Выйти из команды?")) {
      return;
    }
    setBusy(true);
    try {
      await leaveTeam(teamId);
      router.push("/teams");
    } catch (err) {
      const msg =
        err instanceof ApiError && err.code === "team.cannot_leave_as_only_lead"
          ? "Сначала передай роль тимлида другому участнику или архивируй команду."
          : err instanceof ApiError
            ? err.message
            : "Не удалось выйти.";
      window.alert(msg);
      setBusy(false);
    }
  }

  async function handleSubmitLog(e: React.FormEvent) {
    e.preventDefault();
    if (logBusy) return;
    const text = logText.trim();
    if (text.length < 10) {
      window.alert("Минимум 10 символов.");
      return;
    }
    setLogBusy(true);
    try {
      const today = new Date().toISOString().slice(0, 10);
      const res = await submitDailyLog(teamId, {
        log_date: today,
        text_content: text,
        external_url: logUrl || undefined,
        client_source: "web",
      });
      setStreak(res.streak);
      setLogText("");
      setLogUrl("");
      const weekAgo = new Date(Date.now() - 6 * 86400000).toISOString().slice(0, 10);
      const fresh = await listDailyLogs(teamId, weekAgo, today);
      setEntries(fresh.entries);
    } catch (err) {
      window.alert(err instanceof ApiError ? err.message : "Не удалось записать лог.");
    } finally {
      setLogBusy(false);
    }
  }

  async function handleFreeze() {
    if (logBusy) return;
    if (!window.confirm("Заморозить сегодняшний день? Стрик сохранится.")) return;
    setLogBusy(true);
    try {
      const today = new Date().toISOString().slice(0, 10);
      await freezeDailyLog(teamId, today);
      const weekAgo = new Date(Date.now() - 6 * 86400000).toISOString().slice(0, 10);
      const fresh = await listDailyLogs(teamId, weekAgo, today);
      setEntries(fresh.entries);
      const s = await getDailyLogStreak(teamId);
      setStreak(s);
    } catch (err) {
      window.alert(err instanceof ApiError ? err.message : "Не удалось заморозить.");
    } finally {
      setLogBusy(false);
    }
  }

  async function handleApprove(proofId: number) {
    try {
      await approveProof(proofId);
      setFeedItems((prev) => prev.map((p) => (p.id === proofId ? { ...p, status: "approved" } : p)));
    } catch (err) {
      window.alert(err instanceof ApiError ? err.message : "Не удалось одобрить.");
    }
  }

  async function handleRejectSubmit(proofId: number) {
    if (rejectBusy) return;
    const text = rejectText.trim();
    if (text.length < 3) {
      window.alert("Комментарий обязателен при отклонении.");
      return;
    }
    setRejectBusy(true);
    try {
      await rejectProof(proofId, text);
      setFeedItems((prev) => prev.map((p) => (p.id === proofId ? { ...p, status: "rejected" } : p)));
      setRejectProofId(null);
      setRejectText("");
    } catch (err) {
      window.alert(err instanceof ApiError ? err.message : "Не удалось отклонить.");
    } finally {
      setRejectBusy(false);
    }
  }

  async function toggleComments(proofId: number) {
    if (expandedProofId === proofId) {
      setExpandedProofId(null);
      return;
    }
    setExpandedProofId(proofId);
    if (!commentsMap[proofId]) {
      try {
        const comments = await getProofComments(proofId);
        setCommentsMap((prev) => ({ ...prev, [proofId]: comments }));
      } catch {
        setCommentsMap((prev) => ({ ...prev, [proofId]: [] }));
      }
    }
  }

  async function handleAddComment(proofId: number) {
    const text = (commentTextMap[proofId] ?? "").trim();
    if (text.length < 1) return;
    setCommentBusyMap((prev) => ({ ...prev, [proofId]: true }));
    try {
      const c = await addProofComment(proofId, text);
      setCommentsMap((prev) => ({ ...prev, [proofId]: [c, ...(prev[proofId] ?? [])] }));
      setCommentTextMap((prev) => ({ ...prev, [proofId]: "" }));
      setFeedItems((prev) => prev.map((p) => (p.id === proofId ? { ...p, comments_count: p.comments_count + 1 } : p)));
    } catch (err) {
      window.alert(err instanceof ApiError ? err.message : "Не удалось отправить комментарий.");
    } finally {
      setCommentBusyMap((prev) => ({ ...prev, [proofId]: false }));
    }
  }

  async function handleToggleAIConsent() {
    if (state.kind !== "ready") return;
    const next = !aiConsent;
    try {
      await setMyAIConsent(teamId, state.detail.my_membership.user_id, next);
      setAiConsent(next);
      setState({
        kind: "ready",
        detail: {
          ...state.detail,
          my_membership: { ...state.detail.my_membership, ai_consent: next },
        },
      });
    } catch (err) {
      window.alert(err instanceof ApiError ? err.message : "Не удалось изменить настройку AI.");
    }
  }

  async function handleAssembleProof() {
    if (assembleBusy) return;
    setAssembleBusy(true);
    try {
      const res = await assembleProofWithAI(teamId);
      setAssembleResult(res);
    } catch (err) {
      window.alert(err instanceof ApiError ? err.message : "Не удалось собрать proof с AI.");
    } finally {
      setAssembleBusy(false);
    }
  }

  async function handleLoadBriefing(userId: number) {
    if (briefingTarget === userId) {
      setBriefingTarget(null);
      setBriefing(null);
      return;
    }
    setBriefingTarget(userId);
    try {
      const b = await getLeadBriefing(teamId, userId);
      setBriefing(b);
    } catch (err) {
      window.alert(err instanceof ApiError ? err.message : "Не удалось загрузить брифинг.");
      setBriefingTarget(null);
    }
  }

  return (
    <div className={`page-shell ${styles.page}`}>
      <Link href="/teams" className={styles.back}>
        ← КОМАНДЫ
      </Link>

      <header className={styles.header}>
        <h1 className={styles.title}>{team.name.toUpperCase()}</h1>
        <div className={styles.meta}>
          <span className={styles.aiMode}>AI: {aiModeLabel(team.ai_mode)}</span>
          <span className={styles.dot}>·</span>
          <span>{detail.member_count} чел.</span>
          <span className={styles.dot}>·</span>
          <span className={styles.role}>{roleLabel(detail.my_membership.role)}</span>
          {team.archived_at && <span className={styles.archived}>· В АРХИВЕ</span>}
        </div>
      </header>

      {/* Daily Log Section */}
      <section className={styles.section}>
        <div className={styles.sectionLabel}>ЕЖЕДНЕВНЫЙ ЛОГ</div>
        {streak && (
          <div className={styles.streakRow}>
            <span className={styles.streakValue}>🔥 {streak.current}</span>
            <span className={styles.streakLabel}>
              {streak.is_new_record ? "Новый рекорд!" : `Лучший: ${streak.best}`}
            </span>
            <span className={styles.freezeLabel}>
              Заморозок: {streak.freezes_used_this_month}/2
            </span>
          </div>
        )}
        <form onSubmit={handleSubmitLog} className={styles.logForm}>
          <textarea
            className={styles.logInput}
            placeholder="Что нового узнал сегодня? Минимум 10 символов."
            value={logText}
            onChange={(e) => setLogText(e.target.value)}
            rows={3}
            disabled={logBusy}
          />
          <input
            className={styles.urlInput}
            type="url"
            placeholder="Ссылка или скрин (необязательно)"
            value={logUrl}
            onChange={(e) => setLogUrl(e.target.value)}
            disabled={logBusy}
          />
          <div className={styles.logActions}>
            <button type="submit" className={styles.primaryBtn} disabled={logBusy}>
              {logBusy ? "ЗАПИСЫВАЕМ…" : "ЗАПИСАТЬ"}
            </button>
            <button
              type="button"
              className={styles.subtleBtn}
              onClick={handleFreeze}
              disabled={logBusy || (streak?.freezes_used_this_month ?? 0) >= 2}
            >
              ЗАМОРОЗИТЬ
            </button>
          </div>
        </form>
        {entries.length > 0 && (
          <ul className={styles.entryList}>
            {entries.map((e) => (
              <li key={e.id} className={styles.entryItem}>
                <span className={entryStatusClass(e.status)}>{entryStatusLabel(e.status)}</span>
                <span className={styles.entryDate}>{e.log_date}</span>
                {e.text_content && (
                  <span className={styles.entryPreview}>{e.text_content.slice(0, 60)}
                    {e.text_content.length > 60 ? "…" : ""}
                  </span>
                )}
                {e.has_artifact && <span className={styles.entryArtifact}>📎</span>}
              </li>
            ))}
          </ul>
        )}
        {team.ai_mode !== "off" && (
          <div className={styles.aiConsentRow}>
            <label className={styles.aiConsentLabel}>
              <input
                type="checkbox"
                checked={!!aiConsent}
                onChange={handleToggleAIConsent}
                disabled={aiConsent === null}
              />
              Разрешить AI использовать мои данные для персонализации
            </label>
          </div>
        )}
      </section>

      {/* AI Assemble Proof Section */}
      {team.ai_mode !== "off" && (
        <section className={styles.section}>
          <div className={styles.sectionLabel}>AI: СОБРАТЬ PROOF</div>
          <button
            type="button"
            className={styles.subtleBtn}
            onClick={handleAssembleProof}
            disabled={assembleBusy}
          >
            {assembleBusy ? "СОБИРАЕМ…" : "СОБРАТЬ С AI"}
          </button>
          {assembleResult && assembleResult.candidates.length > 0 && (
            <ul className={styles.candidateList}>
              {assembleResult.candidates.map((c, idx) => (
                <li key={idx} className={styles.candidateCard}>
                  <div className={styles.candidateHeader}>
                    <span className={styles.candidateConfidence}>{c.confidence.toUpperCase()}</span>
                    <span className={styles.candidateGoal}>Цель ID: {c.goal_id}</span>
                  </div>
                  <p className={styles.candidateRationale}>{c.rationale}</p>
                  <span className={styles.candidateNotes}>Заметки: {c.note_ids.join(", ")}</span>
                </li>
              ))}
            </ul>
          )}
          {assembleResult && assembleResult.candidates.length === 0 && (
            <div className={styles.emptyCandidates}>AI не предложил кандидатов. Попробуй собрать вручную.</div>
          )}
        </section>
      )}

      {/* Team Feed Section */}
      <section className={styles.section}>
        <div className={styles.sectionLabel}>ЛЕНТА КОМАНДЫ</div>
        {feedItems.length === 0 && !feedBusy && (
          <div className={styles.emptyFeed}>Пока нет доказательств. Скоро появятся.</div>
        )}
        {feedItems.length > 0 && (
          <ul className={styles.feedList}>
            {feedItems.map((item) => (
              <li key={item.id} className={styles.feedCard}>
                <div className={styles.feedHeader}>
                  <span className={styles.feedAlias}>{item.owner_alias}</span>
                  <span className={styles.feedGoal}>{item.goal_title}</span>
                  <span className={styles.feedMeta}>{new Date(item.submitted_at).toLocaleDateString("ru-RU")}</span>
                </div>
                <div className={styles.feedStatusRow}>
                  <span className={feedStatusClass(item.status)}>{feedStatusLabel(item.status)}</span>
                  {item.comments_count > 0 && (
                    <button
                      type="button"
                      className={styles.commentToggle}
                      onClick={() => toggleComments(item.id)}
                    >
                      {item.comments_count} 💬
                    </button>
                  )}
                  {item.comments_count === 0 && (
                    <button
                      type="button"
                      className={styles.commentToggle}
                      onClick={() => toggleComments(item.id)}
                    >
                      💬
                    </button>
                  )}
                </div>
                {item.can_approve && item.status === "submitted" && (
                  <div className={styles.feedActions}>
                    <button
                      type="button"
                      className={styles.approveBtn}
                      onClick={() => handleApprove(item.id)}
                      disabled={busy}
                    >
                      ОДОБРИТЬ
                    </button>
                    <button
                      type="button"
                      className={styles.rejectBtn}
                      onClick={() => {
                        setRejectProofId(item.id);
                        setRejectText("");
                      }}
                      disabled={busy}
                    >
                      ОТКЛОНИТЬ
                    </button>
                  </div>
                )}
                {rejectProofId === item.id && (
                  <div className={styles.rejectForm}>
                    <textarea
                      className={styles.rejectInput}
                      placeholder="Причина отклонения (обязательно)"
                      value={rejectText}
                      onChange={(e) => setRejectText(e.target.value)}
                      rows={2}
                      disabled={rejectBusy}
                    />
                    <div className={styles.rejectActions}>
                      <button
                        type="button"
                        className={styles.rejectBtn}
                        onClick={() => handleRejectSubmit(item.id)}
                        disabled={rejectBusy}
                      >
                        {rejectBusy ? "ОТКЛОНЯЕМ…" : "ОТКЛОНИТЬ"}
                      </button>
                      <button
                        type="button"
                        className={styles.subtleBtn}
                        onClick={() => setRejectProofId(null)}
                        disabled={rejectBusy}
                      >
                        ОТМЕНА
                      </button>
                    </div>
                  </div>
                )}
                {expandedProofId === item.id && (
                  <div className={styles.commentPanel}>
                    {(commentsMap[item.id] ?? []).length > 0 && (
                      <ul className={styles.commentList}>
                        {(commentsMap[item.id] ?? []).map((c) => (
                          <li key={c.id} className={styles.commentItem}>
                            <span className={styles.commentAlias}>{c.user_alias}</span>
                            <span className={styles.commentText}>{c.text}</span>
                            <span className={styles.commentDate}>
                              {new Date(c.created_at).toLocaleDateString("ru-RU")}
                            </span>
                          </li>
                        ))}
                      </ul>
                    )}
                    <div className={styles.commentForm}>
                      <input
                        className={styles.commentInput}
                        type="text"
                        placeholder="Написать комментарий…"
                        value={commentTextMap[item.id] ?? ""}
                        onChange={(e) =>
                          setCommentTextMap((prev) => ({
                            ...prev,
                            [item.id]: e.target.value,
                          }))
                        }
                        onKeyDown={(e) => {
                          if (e.key === "Enter") handleAddComment(item.id);
                        }}
                        disabled={commentBusyMap[item.id]}
                      />
                      <button
                        type="button"
                        className={styles.primaryBtn}
                        onClick={() => handleAddComment(item.id)}
                        disabled={commentBusyMap[item.id]}
                      >
                        ➤
                      </button>
                    </div>
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
        {feedCursor !== null && (
          <button
            type="button"
            className={styles.subtleBtn}
            onClick={() => loadFeed(feedCursor)}
            disabled={feedBusy}
          >
            {feedBusy ? "ЗАГРУЗКА…" : "ЕЩЁ"}
          </button>
        )}
      </section>

      {isLead && (
        <section className={styles.section}>
          <div className={styles.sectionLabel}>КОД ПРИГЛАШЕНИЯ</div>
          <div className={styles.inviteRow}>
            <code className={styles.code}>{team.invite_code}</code>
            <button type="button" className={styles.copyBtn} onClick={handleCopy}>
              {copied ? "СКОПИРОВАНО ✓" : "СКОПИРОВАТЬ"}
            </button>
          </div>
          <button
            type="button"
            className={styles.subtleBtn}
            onClick={handleRegenerate}
            disabled={busy}
          >
            ПЕРЕГЕНЕРИРОВАТЬ КОД
          </button>
        </section>
      )}

      <section className={styles.section}>
        <div className={styles.sectionLabel}>ДЕЙСТВИЯ</div>
        {isLead ? (
          <button
            type="button"
            className={styles.dangerBtn}
            onClick={handleArchive}
            disabled={busy || !!team.archived_at}
          >
            {team.archived_at ? "УЖЕ В АРХИВЕ" : "АРХИВИРОВАТЬ КОМАНДУ"}
          </button>
        ) : (
          <button
            type="button"
            className={styles.dangerBtn}
            onClick={handleLeave}
            disabled={busy}
          >
            ВЫЙТИ ИЗ КОМАНДЫ
          </button>
        )}
      </section>

      {(isLead || detail.my_membership.role === "trusted_approver") && team.ai_mode !== "off" && (
        <section className={styles.section}>
          <div className={styles.sectionLabel}>AI: БРИФИНГ УЧАСТНИКА</div>
          <div className={styles.briefingInputRow}>
            <input
              className={styles.briefingInput}
              type="number"
              placeholder="ID участника"
              value={briefingTarget ?? ""}
              onChange={(e) => setBriefingTarget(Number(e.target.value) || null)}
            />
            <button
              type="button"
              className={styles.subtleBtn}
              onClick={() => {
                if (briefingTarget) handleLoadBriefing(briefingTarget);
              }}
              disabled={!briefingTarget}
            >
              ЗАГРУЗИТЬ БРИФИНГ
            </button>
          </div>
          {briefing && (
            <div className={styles.briefingBox}>
              <p className={styles.briefingText}>{briefing.text}</p>
              {briefing.provider === "kimi" && <span className={styles.aiBadge}>✨ AI</span>}
            </div>
          )}
        </section>
      )}

      <section className={styles.note}>
        Подробный список участников и их прогресс появится в фазе 2.
        Сейчас доступны только базовые операции — создать, пригласить, выйти.
      </section>
    </div>
  );
}

function roleLabel(role: TeamDetail["my_membership"]["role"]): string {
  switch (role) {
    case "lead":
      return "ТИМЛИД";
    case "trusted_approver":
      return "ДОВЕРЕННЫЙ";
    case "member":
      return "УЧАСТНИК";
  }
}

function aiModeLabel(mode: TeamDetail["team"]["ai_mode"]): string {
  switch (mode) {
    case "off":
      return "ВЫКЛ";
    case "metadata-only":
      return "ТОЛЬКО МЕТАДАННЫЕ";
    case "full":
      return "ПОЛНЫЙ";
  }
}

function entryStatusLabel(status: DailyLogEntry["status"]): string {
  switch (status) {
    case "logged":
      return "✓";
    case "frozen":
      return "❄";
    case "skipped":
      return "—";
    case "missed":
      return "✗";
  }
}

function entryStatusClass(status: DailyLogEntry["status"]): string {
  switch (status) {
    case "logged":
      return styles.statusLogged;
    case "frozen":
      return styles.statusFrozen;
    case "skipped":
      return styles.statusSkipped;
    case "missed":
      return styles.statusMissed;
  }
}

function feedStatusLabel(status: FeedItem["status"]): string {
  switch (status) {
    case "submitted":
      return "НА РАССМОТРЕНИИ";
    case "approved":
      return "ОДОБРЕНО";
    case "rejected":
      return "ОТКЛОНЕНО";
    default:
      return status;
  }
}

function feedStatusClass(status: FeedItem["status"]): string {
  switch (status) {
    case "submitted":
      return styles.feedStatusPending;
    case "approved":
      return styles.feedStatusApproved;
    case "rejected":
      return styles.feedStatusRejected;
    default:
      return "";
  }
}

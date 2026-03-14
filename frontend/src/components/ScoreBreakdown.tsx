"use client";

import type { UserAnswer } from "@/lib/types";

interface ScoreBreakdownProps {
  answer: UserAnswer;
}

function scoreColor(score: number): string {
  if (score >= 0.7) return "text-green-600";
  if (score >= 0.4) return "text-yellow-600";
  return "text-red-600";
}

function barColor(score: number): string {
  if (score >= 0.7) return "bg-green-500";
  if (score >= 0.4) return "bg-yellow-500";
  return "bg-red-500";
}

function ProgressBar({ label, score }: { label: string; score: number }) {
  const pct = Math.round(score * 100);
  return (
    <div>
      <div className="mb-1 flex items-center justify-between text-sm">
        <span className="font-medium text-gray-700">{label}</span>
        <span className={`font-semibold ${scoreColor(score)}`}>{pct}%</span>
      </div>
      <div className="h-2.5 w-full overflow-hidden rounded-full bg-gray-200">
        <div
          className={`h-full rounded-full transition-all ${barColor(score)}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

export default function ScoreBreakdown({ answer }: ScoreBreakdownProps) {
  const overall = answer.overallScore ?? 0;
  const overallPct = Math.round(overall * 100);

  const hasDetailedScores =
    answer.semanticScore != null ||
    answer.keywordScore != null ||
    answer.completenessScore != null;

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
      {/* Overall score */}
      <div className="mb-6 text-center">
        <p className="mb-1 text-sm font-medium text-gray-500">Overall Score</p>
        <p className={`text-4xl font-bold ${scoreColor(overall)}`}>
          {overallPct}%
        </p>
        {answer.isCorrect != null && (
          <p
            className={`mt-1 text-sm font-medium ${
              answer.isCorrect ? "text-green-600" : "text-red-600"
            }`}
          >
            {answer.isCorrect ? "Correct" : "Incorrect"}
          </p>
        )}
      </div>

      {/* Detailed score breakdown */}
      {hasDetailedScores && (
        <div className="mb-6 space-y-4">
          <h4 className="text-sm font-semibold text-gray-900">
            Score Breakdown
          </h4>
          {answer.semanticScore != null && (
            <ProgressBar
              label="Semantic (50%)"
              score={answer.semanticScore}
            />
          )}
          {answer.completenessScore != null && (
            <ProgressBar
              label="Completeness (30%)"
              score={answer.completenessScore}
            />
          )}
          {answer.keywordScore != null && (
            <ProgressBar
              label="Keyword (20%)"
              score={answer.keywordScore}
            />
          )}
        </div>
      )}

      {/* Feedback */}
      {answer.feedback && (
        <div className="rounded-lg bg-gray-50 p-4">
          <h4 className="mb-2 text-sm font-semibold text-gray-900">
            Feedback
          </h4>
          <p className="text-sm leading-relaxed text-gray-700 whitespace-pre-line">
            {answer.feedback}
          </p>
        </div>
      )}
    </div>
  );
}

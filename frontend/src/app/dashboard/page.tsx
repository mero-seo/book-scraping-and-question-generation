"use client";

import { useState, useEffect } from "react";
import { getUserStats } from "@/lib/api";
import type { UserStats } from "@/lib/types";

const bloomLabels: Record<string, string> = {
  remember: "Remember",
  understand: "Understand",
  apply: "Apply",
  analyze: "Analyze",
  evaluate: "Evaluate",
  create: "Create",
};

const difficultyLabels: Record<string, string> = {
  easy: "Easy",
  medium: "Medium",
  hard: "Hard",
};

export default function DashboardPage() {
  const [stats, setStats] = useState<UserStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const data = await getUserStats();
        setStats(data);
      } catch {
        // Not logged in
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  if (loading) return <div className="text-center py-12">Loading...</div>;
  if (!stats)
    return (
      <div className="text-center py-12 text-gray-500">
        Please log in to view your dashboard
      </div>
    );

  function scoreColor(score: number) {
    if (score >= 75) return "text-green-600";
    if (score >= 50) return "text-yellow-600";
    return "text-red-600";
  }

  function barWidth(score: number) {
    return `${Math.min(100, Math.max(0, score))}%`;
  }

  function barColor(score: number) {
    if (score >= 75) return "bg-green-500";
    if (score >= 50) return "bg-yellow-500";
    return "bg-red-500";
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Your Dashboard</h1>

      {/* Overview cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-500">Questions Answered</p>
          <p className="text-3xl font-bold">{stats.totalAnswered}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-500">Average Score</p>
          <p className={`text-3xl font-bold ${scoreColor(stats.averageScore)}`}>
            {stats.averageScore.toFixed(1)}
          </p>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-500">Correct Rate</p>
          <p className={`text-3xl font-bold ${scoreColor(stats.correctRate)}`}>
            {stats.correctRate.toFixed(1)}%
          </p>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-500">Avg Time/Question</p>
          <p className="text-3xl font-bold">
            {stats.averageTimeTaken.toFixed(0)}s
          </p>
        </div>
      </div>

      {/* Score by Bloom's level */}
      {Object.keys(stats.scoreByBloom).length > 0 && (
        <div className="bg-white rounded-lg shadow p-6 mb-6">
          <h2 className="text-lg font-semibold mb-4">
            Score by Bloom&apos;s Taxonomy
          </h2>
          <div className="space-y-3">
            {Object.entries(bloomLabels).map(([key, label]) => {
              const score = stats.scoreByBloom[key] || 0;
              return (
                <div key={key}>
                  <div className="flex justify-between text-sm mb-1">
                    <span>{label}</span>
                    <span className={scoreColor(score)}>
                      {score.toFixed(1)}
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div
                      className={`h-2 rounded-full ${barColor(score)}`}
                      style={{ width: barWidth(score) }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Score by difficulty */}
      {Object.keys(stats.scoreByDifficulty).length > 0 && (
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Score by Difficulty</h2>
          <div className="space-y-3">
            {Object.entries(difficultyLabels).map(([key, label]) => {
              const score = stats.scoreByDifficulty[key] || 0;
              return (
                <div key={key}>
                  <div className="flex justify-between text-sm mb-1">
                    <span>{label}</span>
                    <span className={scoreColor(score)}>
                      {score.toFixed(1)}
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div
                      className={`h-2 rounded-full ${barColor(score)}`}
                      style={{ width: barWidth(score) }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

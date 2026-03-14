"use client";

import { useState, useEffect, use } from "react";
import { getAnswerHistory } from "@/lib/api";
import type { UserAnswer } from "@/lib/types";
import ScoreBreakdown from "@/components/ScoreBreakdown";
import Pagination from "@/components/Pagination";

export default function ResultsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [answers, setAnswers] = useState<UserAnswer[]>([]);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const res = await getAnswerHistory({ page, limit: 10 });
        setAnswers(res.data || []);
        setTotalPages(res.pagination.total_pages);
      } catch {
        setAnswers([]);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [id, page]);

  if (loading) return <div className="text-center py-12">Loading...</div>;

  return (
    <div className="max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Answer History</h1>

      {answers.length === 0 ? (
        <p className="text-center text-gray-500 py-8">No answers yet</p>
      ) : (
        <div className="space-y-4">
          {answers.map((answer) => (
            <div key={answer.id} className="bg-white rounded-lg shadow p-4">
              <div className="flex justify-between items-center mb-2">
                <span className="text-sm text-gray-500">
                  {new Date(answer.createdAt).toLocaleString()}
                </span>
                {answer.timeTaken && (
                  <span className="text-xs text-gray-400">
                    {answer.timeTaken}s
                  </span>
                )}
              </div>
              <p className="text-sm mb-3 text-gray-700">
                {answer.answerText.length > 200
                  ? answer.answerText.slice(0, 200) + "..."
                  : answer.answerText}
              </p>
              <ScoreBreakdown answer={answer} />
            </div>
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="mt-6">
          <Pagination
            page={page}
            totalPages={totalPages}
            onPageChange={setPage}
          />
        </div>
      )}
    </div>
  );
}

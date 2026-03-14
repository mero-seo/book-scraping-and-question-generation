"use client";

import { useState, useEffect } from "react";
import { getDashboard } from "@/lib/api";

export default function AdminDashboardPage() {
  const [data, setData] = useState<Record<string, unknown> | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const res = await getDashboard();
        setData(res);
      } catch {
        // Error
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  if (loading) return <div className="text-center py-12">Loading...</div>;
  if (!data) return <div className="text-gray-500">Failed to load dashboard</div>;

  const books = data.books as Record<string, number> | undefined;
  const answers = data.answers as Record<string, unknown> | undefined;

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Admin Dashboard</h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-500">Total Books</p>
          <p className="text-3xl font-bold">{books?.total ?? 0}</p>
          <div className="flex gap-3 mt-2 text-xs text-gray-500">
            <span className="text-green-600">{books?.ready ?? 0} ready</span>
            <span className="text-yellow-600">{books?.processing ?? 0} processing</span>
            <span className="text-red-600">{books?.failed ?? 0} failed</span>
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-500">Chapters</p>
          <p className="text-3xl font-bold">{String(data.chapters ?? 0)}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-500">Questions</p>
          <p className="text-3xl font-bold">{String(data.questions ?? 0)}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-500">Users</p>
          <p className="text-3xl font-bold">{String(data.users ?? 0)}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-500">Total Answers</p>
          <p className="text-3xl font-bold">{String((answers?.total as number) ?? 0)}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <p className="text-sm text-gray-500">Avg Score</p>
          <p className="text-3xl font-bold">
            {((answers?.averageScore as number) ?? 0).toFixed(1)}
          </p>
        </div>
      </div>
    </div>
  );
}

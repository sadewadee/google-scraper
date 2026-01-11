import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/UI/Table"
import { Badge } from "@/components/UI/Badge"
import type { Job } from "@/api/types"

// Temporary mock data
const recentJobs: Job[] = [
    { id: 1, keyword: "coffee shop jakarta", status: "completed", created_at: "2024-01-20T10:00:00Z", updated_at: "2024-01-20T10:05:00Z", result_count: 50 },
    { id: 2, keyword: "coworking space bandung", status: "processing", created_at: "2024-01-20T10:10:00Z", updated_at: "2024-01-20T10:11:00Z" },
    { id: 3, keyword: "florist surabaya", status: "pending", created_at: "2024-01-20T10:15:00Z", updated_at: "2024-01-20T10:15:00Z" },
    { id: 4, keyword: "gym yoga bali", status: "failed", created_at: "2024-01-19T15:00:00Z", updated_at: "2024-01-19T15:05:00Z" },
    { id: 5, keyword: "restaurant padang", status: "completed", created_at: "2024-01-19T14:00:00Z", updated_at: "2024-01-19T14:10:00Z", result_count: 120 },
]

export function RecentJobs() {
    return (
        <div className="rounded-md border bg-card text-card-foreground">
            <div className="p-6">
                <h3 className="text-lg font-medium leading-none tracking-tight">Recent Jobs</h3>
                <p className="text-sm text-muted-foreground mt-2">
                    Latest scraping jobs and their status.
                </p>
            </div>
            <div className="relative w-full overflow-auto">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="w-[100px]">ID</TableHead>
                            <TableHead>Keyword</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead className="text-right">Results</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {recentJobs.map((job) => (
                            <TableRow key={job.id}>
                                <TableCell className="font-medium">{job.id}</TableCell>
                                <TableCell>{job.keyword}</TableCell>
                                <TableCell>
                                    <Badge variant={job.status}>{job.status}</Badge>
                                </TableCell>
                                <TableCell className="text-right">{job.result_count || "-"}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>
        </div>
    )
}

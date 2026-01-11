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

interface RecentJobsProps {
    jobs: Job[]
}

export function RecentJobs({ jobs }: RecentJobsProps) {
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
                        {jobs.map((job) => (
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

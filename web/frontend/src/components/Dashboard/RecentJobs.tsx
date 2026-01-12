import { Link } from "react-router-dom"
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
                            <TableHead>Name</TableHead>
                            <TableHead>Keywords</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead className="text-right">Progress</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {jobs.map((job) => (
                            <TableRow key={job.id}>
                                <TableCell className="font-medium">
                                    <Link to={`/jobs/${job.id}`} className="hover:underline">
                                        {job.name}
                                    </Link>
                                </TableCell>
                                <TableCell className="text-muted-foreground text-xs max-w-[150px] truncate">
                                    {job.config.keywords.join(", ")}
                                </TableCell>
                                <TableCell>
                                    <Badge variant={job.status}>{job.status}</Badge>
                                </TableCell>
                                <TableCell className="text-right">
                                    {job.progress.scraped_places}/{job.progress.total_places || "-"}
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>
        </div>
    )
}

import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Link } from "react-router-dom"
import { jobsApi } from "@/api/jobs"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/UI/Table"
import { Badge } from "@/components/UI/Badge"
import { Button } from "@/components/UI/Button"
import { Input } from "@/components/UI/Input"
import {
    Trash2,
    XCircle,
    Search,
    Eye
} from "lucide-react"

export function JobTable() {
    const [searchTerm, setSearchTerm] = useState("")
    const queryClient = useQueryClient()

    const { data: response, isLoading, error } = useQuery({
        queryKey: ["jobs"],
        queryFn: jobsApi.getAll,
    })

    const deleteMutation = useMutation({
        mutationFn: jobsApi.delete,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["jobs"] })
        }
    })

    const cancelMutation = useMutation({
        mutationFn: jobsApi.cancel,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["jobs"] })
        }
    })

    const jobs = response?.data || []
    const filteredJobs = jobs.filter(job =>
        job.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        job.config.keywords.some(k => k.toLowerCase().includes(searchTerm.toLowerCase()))
    )

    return (
        <div className="space-y-4">
            <div className="flex items-center gap-2 max-w-sm">
                <Search className="h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder="Search jobs..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="max-w-sm"
                />
            </div>

            {error && (
                <div className="text-destructive text-sm p-4 bg-destructive/10 rounded-md">
                    Failed to load jobs: {error instanceof Error ? error.message : "Unknown error"}
                </div>
            )}

            <div className="rounded-md bg-neu-base shadow-neu-flat">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Keywords</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Priority</TableHead>
                            <TableHead className="text-right">Progress</TableHead>
                            <TableHead>Created At</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={7} className="h-24 text-center">
                                    Loading jobs...
                                </TableCell>
                            </TableRow>
                        ) : filteredJobs.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={7} className="h-24 text-center">
                                    No jobs found.
                                </TableCell>
                            </TableRow>
                        ) : (
                            filteredJobs.map((job) => (
                                <TableRow key={job.id}>
                                    <TableCell className="font-medium">
                                        <Link to={`/jobs/${job.id}`} className="hover:underline">
                                            {job.name}
                                        </Link>
                                    </TableCell>
                                    <TableCell className="text-muted-foreground text-xs max-w-[200px] truncate">
                                        {job.config.keywords.join(", ")}
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant={job.status}>{job.status}</Badge>
                                    </TableCell>
                                    <TableCell>
                                        <span className={`text-xs ${job.priority >= 8 ? 'text-red-500 font-bold' :
                                            job.priority <= 3 ? 'text-gray-500' : 'text-foreground'
                                            }`}>
                                            {job.priority}
                                        </span>
                                    </TableCell>
                                    <TableCell className="text-right">
                                        {job.progress.scraped_places}/{job.progress.total_places || "-"}
                                        {job.progress.percentage > 0 && (
                                            <span className="text-muted-foreground ml-1">
                                                ({Math.round(job.progress.percentage)}%)
                                            </span>
                                        )}
                                    </TableCell>
                                    <TableCell className="text-muted-foreground text-xs">
                                        {new Date(job.created_at).toLocaleDateString()}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-1">
                                            <Button variant="ghost" size="icon" asChild title="View">
                                                <Link to={`/jobs/${job.id}`}>
                                                    <Eye className="h-4 w-4" />
                                                </Link>
                                            </Button>

                                            {(job.status === 'pending' || job.status === 'running') && (
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    title="Cancel"
                                                    onClick={() => cancelMutation.mutate(job.id)}
                                                    disabled={cancelMutation.isPending}
                                                >
                                                    <XCircle className="h-4 w-4" />
                                                </Button>
                                            )}

                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                className="text-destructive hover:text-destructive"
                                                title="Delete"
                                                onClick={() => deleteMutation.mutate(job.id)}
                                                disabled={deleteMutation.isPending}
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </Button>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            <div className="flex items-center justify-end space-x-2 py-4">
                <Button variant="outline" size="sm" disabled>Previous</Button>
                <Button variant="outline" size="sm" disabled>Next</Button>
            </div>
        </div>
    )
}

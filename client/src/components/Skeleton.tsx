const SkeletonRow = ({ cols = 5 }: { cols?: number }) => (
  <tr className="animate-pulse">
    {Array.from({ length: cols }).map((_, i) => (
      <td key={i} className="py-3 px-4">
        <div className="h-4 bg-[#e9edff] rounded w-3/4" />
      </td>
    ))}
  </tr>
)

export const SkeletonCard = () => (
  <div className="animate-pulse p-4 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm">
    <div className="h-4 bg-[#e9edff] rounded w-1/2 mb-3" />
    <div className="h-8 bg-[#e9edff] rounded w-1/3 mb-2" />
    <div className="h-3 bg-[#e9edff] rounded w-2/3" />
  </div>
)

export default SkeletonRow

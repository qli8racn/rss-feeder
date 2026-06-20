import { categoryStyle } from '../../domain/category'

interface CategoryBadgeProps {
  category: string
  className?: string
}

function CategoryBadge({ category, className = '' }: CategoryBadgeProps) {
  const style = categoryStyle(category)
  return (
    <span
      className={`rounded text-caption ${className}`}
      style={{ color: style.text, backgroundColor: style.bg }}
    >
      {category}
    </span>
  )
}

export default CategoryBadge
